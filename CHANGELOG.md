# Changelog

Todas as alteracoes notaveis neste projeto serao documentadas neste arquivo.
O formato e baseado em [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/) e este projeto adere ao [Versionamento Semantico](https://semver.org/lang/pt-BR/).

## [v0.5.0] : 2026-10-16

### Adicionado

- Arquitetura autônoma em Go nativo com o padrão de Ator (`RoomActor`), isolando cada sala em sua própria goroutine com canal inbox thread-safe.
- Conexão bidirecional via WebSockets full-duplex (`/ws/salas/{codigo}`) com streaming contínuo do slider analógico (30 a 60 Hz) e envelopes tipados.
- Gerenciador central de salas em memória RAM (`LobbyHub`), eliminando persistência em disco ou bancos relacionais.
- Moderação autoritativa pelo anfitrião distinguindo expulsão (desocupação imediata de vaga) e banimento persistente da sala por `DeviceID`.
- Arbitragem concorrente de Pedra, Papel e Tesoura em memória e catálogo temático embutido de cartas de espectros.
- Suporte flexível a flags e ao subcomando `serve` no ponto de entrada (`cmd/server/main.go`).

### Removido

- Dependências, schemas (`pb_schema.json`) e hooks legados do PocketBase (`internal/hooks/`).
- Rotas REST síncronas e difusão unidirecional via Server-Sent Events (SSE).

## [v0.3.1] : 2026-10-15

### Adicionado

- Implementação autoritativa do endpoint de moderação `POST /api/leveling/salas/{codigo}/expulsar` com validação de titularidade do anfitrião (`host_id`), checagem de `session_token`, bloqueio de auto-expulsão e deleção atômica com broadcast SSE.
- Renomeação formal e completa das equipes no domínio de backend para `PATO` e `GANSO` (`EquipePato` e `EquipeGanso`), expurgando referências legadas.

## [v0.3.0] : 2026-10-14

### Adicionado

- Rota autoritativa de monitoramento de presenca periodica em `/api/leveling/salas/{codigo}/heartbeat` com validacao de `session_token` e atualizacao do carimbo `last_seen`.
- Rota autoritativa de selecao e troca de equipes em `/api/leveling/salas/{codigo}/equipe` com limite estrito de 2 integrantes para Ganso e Pato e emissao em tempo real via SSE.
- Rota soberana de dissolucao de sala pelo anfitriao em `/api/leveling/salas/{codigo}/desfazer`.
- Divisao tripartite de equipes (Aleatorio, Pato, Ganso), controle de quorum de 4 jogadores e validacao de inicio de partida exclusivo pelo anfitriao.
- Especificacao formal do contrato e governanca para a rota autoritativa de moderacao e expulsao de participantes (`POST /api/leveling/salas/{codigo}/expulsar`) em `CONTRACT.md` e `AGENTS.md`.
- Campos `session_token` na colecao `jogadores` e `motivo_fim` na colecao `salas`, sincronizados de forma idempotente no bootstrap.

### Corrigido

- Resolucao de erro interno 500 no endpoint de ingresso (`/api/leveling/salas/{codigo}/entrar`) ao migrar consultas com filtros em string para `FindAllRecords` com `dbx.HashExp` compativel com PocketBase Go SDK v0.24+.
- Ajuste nas rotinas periodicas de expurgo em `cleanup.go` utilizando consultas parametrizadas seguras no banco SQLite.

## [v0.2.0] : 2026-10-13

### Adicionado

- Portabilidade completa das regras de jogo e validacoes autoritativas para Go nativo estendendo o PocketBase.
- Modelagem formal de Entidades de Dominio: `EntidadeSessao`, `EntidadeParticipante` e `EntidadeDisputa`.
- Implementacao de Componentes Atomicos autoritativos: `RegraProximidadeComponent`, `SigiloMetaComponent`, `ArbitroPptComponent` e `CleanupComponent`.
- Rota de leitura de alto desempenho com SQL-Generated JSON (Pass-Through) em `/api/leveling/salas/{codigo}/snapshot`.
- Auto-bootstrap de colecoes via `pb_schema.json` embarcado no binario.
- Pipeline de compilacao cruzada estatica (CGO desabilitado) para Linux, macOS e Windows.

### Alterado

- Substituicao da interpretacao Goja em JavaScript por codigo compilado em Go, eliminando custo de reflexao em tempo de execucao.
- Arbitragem de disputas de iniciativa protegida por primitivas nativas de sincronizacao (`sync.RWMutex`).

## [v0.1.0] : 2026-10-10

### Adicionado

- Prototipo inicial do backend com PocketBase e regras provisorias em JavaScript via `pb_hooks`.
- Minijogo de Pedra, Papel e Tesoura para iniciativa.
- Hooks de expurgacao automatica de salas e conexoes ociosas.
