# Changelog

Todas as alteracoes notaveis neste projeto serao documentadas neste arquivo.
O formato e baseado em [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/) e este projeto adere ao [Versionamento Semantico](https://semver.org/lang/pt-BR/).

## [v0.3.0] : 2026-10-14

### Adicionado

- Rota autoritativa de monitoramento de presenca periodica em `/api/leveling/salas/{codigo}/heartbeat` com validacao de `session_token` e atualizacao do carimbo `last_seen`.
- Rota autoritativa de selecao e troca de equipes em `/api/leveling/salas/{codigo}/equipe` com limite estrito de 2 integrantes para Aguia e Corvo e emissao em tempo real via SSE.
- Rota soberana de dissolucao de sala pelo anfitriao em `/api/leveling/salas/{codigo}/desfazer`.
- Divisao tripartite de equipes (Aleatorio, Corvo, Aguia), controle de quorum de 4 jogadores e validacao de inicio de partida exclusivo pelo anfitriao.
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
