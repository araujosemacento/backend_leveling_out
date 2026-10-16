# AGENTS.md: Diretrizes para Agentes Autônomos (Backend: Leveling Out)

Este documento é a fonte primária de verdade e instrução para qualquer desenvolvedor ou agente autônomo de inteligência artificial que opere neste repositório. Ele descreve o propósito do jogo **Leveling Out**, o fluxo de seu _game loop_, a arquitetura técnica adotada e os padrões estritos de desenvolvimento do backend.

---

## 1. Visão Geral do Jogo: Leveling Out

**Leveling Out** é um _party game_ multiplayer cooperativo-competitivo focado em **dedução social, sintonia cultural e calibração conceitual**. Duas equipes competem tentando posicionar e adivinhar conceitos abstratos ao longo de uma escala contínua de 0% a 100%.

### Os 3 Pilares da Jogabilidade

1. **Nível Oculto:** O preenchimento visual de um tubo de ensaio com volume percentual aleatório (0% a 100%), visível **apenas** para o Codificador sorteado da equipe da vez.
2. **Gerar Pistas:** A partir de uma carta com um espectro bipolar (ex.: _"Famoso / Anônimo"_, _"Quente / Frio"_, _"Fácil / Difícil"_), o Codificador formula uma dica conceitual para orientar seu parceiro até o nível exato (ex.: meta de 70% em "Famoso/Anônimo" $\rightarrow$ pista: _"Keanu Reeves"_).
3. **Acerto Aproximado:** O Palpiteiro da equipe desloca um marcador analógico (_slider_) até onde estima estar o líquido. A pontuação é conferida por faixas de proximidade:
   - Margem de erro $\le 2\%$: **Na mosca (+4 pontos)**
   - Margem de erro $\le 6\%$: **Muito perto (+3 pontos)**
   - Margem de erro $\le 12\%$: **Perto (+2 pontos)**
   - Margem de erro $> 12\%$: **Errou (0 pontos)**

---

## 2. Game Loop Detalhado em 3 Fases

### Fase 1: Onboarding, Sala e Decisão de Turno

Os jogadores conectam-se via WebSocket na rota `/ws/salas/{codigo}`. Na sala, os participantes são distribuídos em duas equipes (**Equipe Pato / A** e **Equipe Ganso / B**) ou alocados como indecisos (`ALEATORIO`). O criador da sala é designado como anfitrião (`isHost`), possuindo a prerrogativa autoritativa de moderação para remover participantes indesejados (`EXPULSAR_JOGADOR`) e iniciar a partida após o preenchimento de todas as 4 vagas da sala (`INICIAR_PARTIDA`). Para definir quem começa, o sistema dispara um minijogo sincronizado de **Pedra, Papel e Tesoura** entre os representantes das equipes com limite de 10 segundos. A equipe vencedora ganha o direito de iniciar e seleciona o espectro conceitual da primeira rodada.

### Fase 2: O Core Loop da Rodada (Dica, Palpite e Revelação)

1. O ator da sala instancia a rodada ativa na memória (`RodadaAtiva`) e define os papéis da equipe da vez.
2. É sorteada automaticamente uma `meta_oculta` (inteiro criptográfico entre 5 e 95) via `SecretTargetComponent`.
3. O Codificador formula e envia uma dica textual sanitizada (`SUBMETER_DICA`).
4. Ao receber a dica via broadcast `SYNC`, o Palpiteiro move o slider analógico (transmitindo `SLIDER_MOVER` para visualização em tempo real) e confirma seu `palpite` definitivo (0 a 100).
5. Ocorre a **revelação simultânea**: o ator calcula a distância absoluta entre `meta_oculta` e `palpite`, computa os pontos conquistados via `ProximityComponent`, atualiza o placar macro da sala (`placarA` ou `placarB`) e inicia contagem regressiva para o próximo turno.

### Fase 3: Progressão, Alternância e Conclusão com Revanche

Os turnos alternam estritamente entre as equipes. A partida segue até que uma das equipes atinja a pontuação máxima estipulada (10 pontos). Ao atingir a meta, a fase muda para `VITORIA`, exibindo a tela de celebração com métricas de sintonia da partida baseadas no histórico de rodadas em memória. A sala oferece opção de **Revanche** (`REVANCHE`), mantendo equipes e zerando o placar.

---

## 3. Arquitetura do Backend e Filosofia de Design

O backend é construído em **Go nativo puro** (`CGO_ENABLED=0`), estruturado sob o padrão de **Ator em Memória** (`RoomActor`) e comunicação bidirecional de baixa latência via **WebSockets**.

### Diretrizes Centrais Inegociáveis

1. **Estado 100% em Memória RAM (Actor Pattern):**
   - Cada sala é uma goroutine independente gerenciando seu estado isolado em memória.
   - Toda comunicação com a sala passa por um canal serializador (`inbox chan RoomMessage`), eliminando condições de corrida sem travas de banco ou overhead de disco.
2. **Consumo Mínimo de Recursos:**
   - O processo completo opera entre **12 MB e 25 MB de RAM**, permitindo execução contínua em servidores modestos sem sobrecarga de heap.
3. **Anonimato e Ausência de Contas:**
   - **NÃO** crie tabelas permanentes de usuários, senhas ou sistemas de OAuth.
   - Cada participante é identificado por um `playerId` (UUID gerado no navegador) e autenticado por `sessionToken` volátil em memória.
4. **Efemeridade e Higienização Automática:**
   - Não há persistência relacional em disco. Salas inativas são encerradas automaticamente e recicladas pelo Garbage Collector do Go.
5. **Comunicação em Tempo Real Bidirecional:**
   - Conexão WebSocket unificada com frames JSON tipados (`internal/protocol/messages.go`), suportando streaming contínuo do slider (30-60 Hz).
6. **Composição por Entidades e Componentes Atômicos:**
   - `EntidadeSessao` (`internal/entities/session.go`): salas e rodadas ativas.
   - `EntidadeParticipante` (`internal/entities/participant.go`): jogadores e equipes Pato e Ganso.
   - `EntidadeDisputa` (`internal/entities/contest.go`): confrontos e lances de PPT.
   - `ProximityComponent` (`internal/components/proximity.go`): algoritmo matemático de pontuação por tolerância.
   - `SecretTargetComponent` (`internal/components/secret_target.go`): segredo seguro [5, 95] com `crypto/rand`.
   - `PptArbiterComponent` (`internal/components/ppt_arbiter.go`): arbitragem determinística de PPT e preenchimento por timeout.
   - `SpectrumsComponent` (`internal/components/spectrums.go`): catálogo de 15 cartas bipolares e sorteio de opções.
7. **Proibição de Commits e Pushes Autônomos:**
   - Agentes de IA são expressamente proibidos de executar `git commit`, `git tag` ou `git push` de forma autônoma.
   - O agente deve unicamente fornecer o sumário estruturado, a descrição detalhada e o comando git sugerido pronto para que o desenvolvedor humano realize a ação manualmente.

---

## 4. Estrutura do Repositório e Papéis dos Arquivos

```text
backend/
├── cmd/
│   └── server/main.go               <-- Ponto de entrada do servidor nativo HTTP/WebSocket
├── internal/
│   ├── actor/                       <-- Padrão de Ator para gerência de salas em memória
│   │   ├── room_actor.go            <-- Goroutine autônoma por sala, inbox chan, timers e Zero Trust
│   │   ├── client_conn.go           <-- Conexão thread-safe de escrita/leitura sobre WebSocket
│   │   └── actor_test.go            <-- Testes unitários do ciclo de vida e projeções
│   ├── hub/
│   │   └── lobby_hub.go             <-- Hub central thread-safe de salas ativas em memória
│   ├── protocol/
│   │   └── messages.go              <-- Contratos tipados de entrada e envelopes de saída
│   ├── entities/                    <-- EntidadeSessao, EntidadeParticipante e EntidadeDisputa
│   │   ├── session.go               <-- SalaEstado, RodadaAtiva, RodadaResumo
│   │   ├── participant.go           <-- Jogador, projeção individual e papéis
│   │   └── contest.go               <-- PptContestState e lances
│   └── components/                  <-- Componentes atômicos autoritativos
│       ├── proximity.go             <-- Regra matemática de proximidade (+4, +3, +2, 0)
│       ├── secret_target.go         <-- Sorteio criptográfico seguro da meta oculta
│       ├── ppt_arbiter.go           <-- Arbitragem determinística de Pedra, Papel e Tesoura
│       ├── spectrums.go             <-- Catálogo de cartas de espectro e sorteio
│       └── components_test.go       <-- Testes unitários com verificação de corrida (-race)
├── scripts/
│   └── start_tunnel.sh              <-- Script utilitário de túnel reverso (Tailscale / Cloudflare)
├── CONTRACT.md                      <-- Especificação formal do protocolo WebSocket e payloads
├── README.md                        <-- Instruções de compilação, execução e operação
├── AGENTS.md                        <-- Este documento (fonte de verdade para agentes de IA)
├── go.mod / go.sum                  <-- Módulos e dependências de compilação Go
└── .gitignore                       <-- Bloqueio de artefatos de build locais
```

---

## 5. Padrões de Desenvolvimento em Go Nativo

- **Zero CGO:** Toda compilação mantém `CGO_ENABLED=0`, garantindo portabilidade estática absoluta entre Linux, macOS e Windows.
- **Concorrência Segura via Actor Pattern:** Nunca compartilhe estado mutável entre requisições. Todas as ações de uma sala transitam pelo canal `inbox` do respectivo `RoomActor`.
- **Zero I/O de Disco no Gameplay:** Operação 100% in-memory, eliminando gargalos de latência de escrita em disco.
- **Buffers de Escrita Assíncronos:** Cada conexão `ClientConn` possui canal interno de escrita, impedindo que conexões lentas de clientes degradem a performance da sala.

---

## 6. Comandos e Testes de Validação

### Executar a suíte de testes unitários do backend com detector de corrida

```bash
go test -v -race ./internal/...
```

### Compilar o executável do servidor localmente

```bash
go build -o leveling-out-server ./cmd/server
```

### Iniciar o servidor

```bash
./leveling-out-server
```

### Testando a saúde da API

```bash
curl -I http://127.0.0.1:8090/health
```
