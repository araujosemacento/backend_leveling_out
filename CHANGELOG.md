# Changelog

Todas as alteracoes notaveis neste projeto serao documentadas neste arquivo.
O formato e baseado em [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/) e este projeto adere ao [Versionamento Semantico](https://semver.org/lang/pt-BR/).

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
