# Leveling Out: Servidor Realtime Go Nativo em Memória

Este diretório contém o servidor em tempo real e a autoridade de regras de jogo para o multiplayer **Leveling Out**, construído em **Go nativo puro** sob o padrão de **Ator em Memória** (`RoomActor`) e comunicação bidirecional de alta frequência via **WebSockets**.

---

## 1. Princípios de Engenharia

- **Estado 100% Efêmero em RAM:** Nenhuma informação persiste em disco ou banco de dados relacional. Cada sala é uma goroutine independente gerenciando seu estado isolado em memória com canal serializador (`inbox chan RoomMessage`), eliminando condições de corrida e latências de I/O.
- **Transmissão Bidirecional Full-Duplex:** Conexão persistente WebSocket (`/ws/salas/{codigo}`) com protocolo tipado de mensagens, suportando streaming contínuo do slider analógico em tempo real (30-60 Hz).
- **Consumo Mínimo de Recursos:** O processo completo opera entre **12 MB e 25 MB de RAM**, permitindo execução contínua em servidores modestos sem sobrecarga de heap.
- **Anonimato e Zero Trust:** Participantes anônimos identificados por UUID local e autenticados por `sessionToken` volátil em memória. A `meta_oculta` do tubo é blindada pelo servidor e omitida dos broadcasts até a fase de revelação oficial.
- **Portabilidade Estática:** Compilado estaticamente com `CGO_ENABLED=0`, sem dependências externas de runtime C ou interpretadores.

---

## 2. Início Rápido

### Compilação a partir do Código Fonte

Requer Go 1.24+ instalado:

```bash
# Executar a suíte de testes unitários do backend
go test -v ./internal/...

# Compilar o binário autossuficiente
go build -o leveling-out-server ./cmd/server

# Iniciar o servidor
./leveling-out-server
```

Por padrão, o servidor escuta em `0.0.0.0:8090` e disponibiliza:

- **Endpoint WebSocket:** `ws://127.0.0.1:8090/ws/salas/{codigo}`
- **Health Check HTTP:** `http://127.0.0.1:8090/health`

---

## 3. Variáveis de Ambiente

O servidor pode ser configurado através de variáveis de ambiente:

| Variável         | Padrão | Descrição                                                                                               |
| :--------------- | :----- | :------------------------------------------------------------------------------------------------------ |
| `PORT`           | `8090` | Porta TCP em que o servidor HTTP/WebSocket escuta.                                                      |
| `ALLOWED_ORIGIN` | `*`    | Origem autorizada no CORS para conexão de navegadores (ex.: `https://meujogo.com`).                     |
| `PROXY_SECRET`   | `""`   | Segredo de infraestrutura para validar conexões encaminhadas por túneis reversos ou proxies Cloudflare. |

Exemplo de execução configurada:

```bash
PORT=8090 ALLOWED_ORIGIN="http://localhost:5173" ./leveling-out-server
```

---

## 4. Exposição para a Internet (Túnel Reverso)

Para permitir que participantes em redes externas ou dispositivos móveis (4G/5G) ingressem na partida:

### Opção A: Tailscale Funnel (Recomendado se já usa Tailscale)

```bash
tailscale funnel 8090
```

Seu servidor ficará acessível publicamente com terminação TLS em:
`wss://sua-maquina.seu-tailnet.ts.net/ws/salas/{codigo}`

### Opção B: Cloudflare Tunnel (cloudflared)

```bash
cloudflared tunnel --url http://127.0.0.1:8090
```

---

## 5. Executando em Segundo Plano (systemd no Linux)

Crie o arquivo `/etc/systemd/system/leveling-out.service`:

```ini
[Unit]
Description=Leveling Out Realtime Go Server
After=network.target

[Service]
Type=simple
User=melo
WorkingDirectory=/home/melo/Documentos/GitHub/leveling_out/backend
ExecStart=/home/melo/Documentos/GitHub/leveling_out/backend/leveling-out-server
Restart=always
RestartSec=3
Environment=PORT=8090
Environment=ALLOWED_ORIGIN=*

[Install]
WantedBy=multi-user.target
```

Ative e inicie o serviço:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now leveling-out
```
