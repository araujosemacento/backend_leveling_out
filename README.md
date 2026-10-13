# Leveling Out: Backend Realtime Autohospedado

Este diretório contém o servidor em tempo real e as regras de domínio para o jogo **Leveling Out**, construído sobre o **PocketBase** (binário compilado em Go com SQLite embarcado).

---

## 1. Princípios de Design

- **Efêmero por Definição:** Nenhuma informação persiste além do ciclo de vida da sala ativa. Usuários são anônimos (sem contas, senhas ou e-mails).
- **Higienização Automática:** O componente nativo autoritativo `CleanupComponent` purga periodicamente salas e registros sem atividade após 15 minutos via cron interno, impedindo acúmulo de dados no disco.
- **Exposição Segura via Túnel Reverso:** O backend roda localmente e é exposto com terminação TLS automática via **Tailscale Funnel** ou **Cloudflare Tunnel**.

---

## 2. Início Rápido

### Opção A: Executar a partir dos Binários Oficiais (GitHub Releases)

Baixe o executável pré-compilado para seu sistema operacional (Linux, macOS ou Windows) na aba de [Releases do Repositório](https://github.com/araujosemacento/backend_leveling_out/releases).

Descompacte e execute:

```bash
# No Linux ou macOS:
./leveling-out-server serve --http="127.0.0.1:8090"

# No Windows:
leveling-out-server.exe serve --http="127.0.0.1:8090"
```

### Opção B: Compilar a partir do Código Fonte em Go

Com o Go 1.24+ instalado:

```bash
# Executar a suíte de testes unitários do backend
go test -v ./internal/components/...

# Compilar o binário autossuficiente
go build -o leveling-out-server ./cmd/server

# Iniciar o servidor
./leveling-out-server serve --http="127.0.0.1:8090"
```

O esquema de banco de dados (`pb_schema.json`) é embarcado diretamente no executável via `//go:embed` e inicializado automaticamente na primeira execução, sem necessidade de configuração manual.

O servidor inicializará e exibirá as URLs:

- **API REST / SSE:** `http://127.0.0.1:8090/api/`
- **Painel Administrativo:** `http://127.0.0.1:8090/_/`

### Acesso ao Painel Administrativo

Ao abrir o painel administrativo (`http://127.0.0.1:8090/_/`) pela primeira vez:

1. Crie sua conta de administrador local (usada exclusivamente para inspecionar registros e conexões);
2. As coleções `salas`, `rodadas` e `jogadores` já estarão criadas e configuradas automaticamente pelo auto-bootstrap nativo do servidor.

---

## 3. Exposição para a Internet (Túnel Reverso)

Para que amigos e dispositivos em redes móveis (4G/5G) consigam se conectar:

### Opção A: Tailscale Funnel (Recomendado se já usa Tailscale)

```bash
tailscale funnel 8090
```

Seu servidor ficará acessível publicamente com HTTPS em:
`https://sua-maquina.seu-tailnet.ts.net/api/`

### Opção B: Cloudflare Tunnel (cloudflared)

```bash
cloudflared tunnel --url http://127.0.0.1:8090
```

A Cloudflare fornecerá uma URL temporária gratuita (ou atrelada a seu domínio próprio) com HTTPS válido.

---

## 4. Executando em Segundo Plano (Indefinidamente)

Para manter o PocketBase rodando na sua máquina mesmo após fechar o terminal:

### Via `systemd` (Linux)

Crie `/etc/systemd/system/pocketbase-levelingout.service`:

```ini
[Unit]
Description=PocketBase Leveling Out Server
After=network.target

[Service]
Type=simple
User=melo
WorkingDirectory=/home/melo/Documentos/GitHub/portfolio_programacao_jogos/backend
ExecStart=/home/melo/Documentos/GitHub/portfolio_programacao_jogos/backend/pocketbase serve --http=127.0.0.1:8090
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

Ative e inicialize:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now pocketbase-levelingout.service
```

---

## 5. Estrutura de Arquivos

```text
backend/
├── AGENTS.md                  <-- Guia detalhado para agentes autônomos de IA
├── CONTRACT.md                <-- Especificação de API, modelos de dados e eventos SSE
├── README.md                  <-- Este guia
├── pb_schema.json             <-- Esquema declarativo das coleções do PocketBase
├── pb_hooks/                  <-- Scripts em JavaScript do runtime Goja
│   ├── cleanup.pb.js          <-- Cron para purga de salas inativas
│   └── game_rules.pb.js       <-- Validações de integridade das fases
├── scripts/
│   ├── download_pocketbase.sh <-- Script de download do binário
│   └── start_tunnel.sh        <-- Script para inicialização do túnel
└── .gitignore                 <-- Proteção contra commit de dados de runtime
```
