# Leveling Out: Backend Realtime Autohospedado

Este diretório contém o servidor em tempo real e as regras de domínio para o jogo **Leveling Out**, construído sobre o **PocketBase** (binário compilado em Go com SQLite embarcado).

---

## 1. Princípios de Design

- **Efêmero por Definição:** Nenhuma informação persiste além do ciclo de vida da sala ativa. Usuários são anônimos (sem contas, senhas ou e-mails).
- **Higienização Automática:** Um hook cron interno (`pb_hooks/cleanup.pb.js`) purga automaticamente salas e registros sem atividade após 15 minutos, impedindo acúmulo de dados no disco.
- **Exposição Segura via Túnel Reverso:** O backend roda localmente e é exposto com terminação TLS automática via **Tailscale Funnel** ou **Cloudflare Tunnel**.

---

## 2. Início Rápido

### Passo 1: Baixar o PocketBase

Você pode usar um dos scripts automatizados conforme seu sistema operacional:

Para Linux ou macOS (Bash):
```bash
chmod +x scripts/download_pocketbase.sh
./scripts/download_pocketbase.sh
```

Para Windows (duplo clique no arquivo ou via CMD/PowerShell):
```cmd
scripts\download_pocketbase.bat
```

Para qualquer sistema operacional com Python 3:
```bash
python3 scripts/download_pocketbase.py
```

Ou acesse as [Releases oficiais do PocketBase](https://github.com/pocketbase/pocketbase/releases) e descompacte o executável `pocketbase` nesta pasta.

### Passo 2: Executar o Servidor

```bash
./pocketbase serve --http="127.0.0.1:8090"
```

O servidor inicializará e exibirá as URLs:

- **API REST / SSE:** `http://127.0.0.1:8090/api/`
- **Painel Administrativo:** `http://127.0.0.1:8090/_/`

### Passo 3: Importar o Esquema de Coleções

Ao abrir o painel administrativo (`http://127.0.0.1:8090/_/`) pela primeira vez:

1. Crie sua conta de administrador local (usada apenas para gerenciar o painel);
2. Vá em **Settings** (ícone de engrenagem) > **Sync** > **Import collections**;
3. Carregue o arquivo [pb_schema.json](./pb_schema.json) para criar automaticamente as coleções `salas` e `jogadores` com todas as regras de acesso e índices pré-configurados.

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

### Via `systemd` (Linux):

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
