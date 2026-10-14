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

Os jogadores acessam a plataforma via navegador estático e entram em uma sala compartilhada via código de sala (`codigo`). Na sala, os participantes são distribuídos em duas equipes (**Equipe Corvo / A** e **Equipe Águia / B**). Para definir quem começa, o sistema dispara um minijogo sincronizado de **Pedra, Papel e Tesoura** entre os capitães das equipes. A equipe vencedora ganha o direito de iniciar e seleciona o espectro conceitual da primeira rodada.

### Fase 2: O Core Loop da Rodada (Dica, Palpite e Revelação)

1. O backend cria um registro na coleção `rodadas` para a rodada ativa da sala e define os papéis da equipe da vez.
2. É sorteada automaticamente uma `meta_oculta` (inteiro entre 5 e 95) armazenada no registro de `rodadas`.
3. O Codificador formula e envia uma `dica` textual.
4. Ao receber a dica via SSE, o Palpiteiro move o slider e confirma seu `palpite` (0 a 100).
5. Ocorre a **revelação simultânea**: o hook calcula a distância absoluta entre `meta_oculta` e `palpite`, computa os pontos conquistados na coleção `rodadas` e atualiza o placar macro da sala (`placar_a` ou `placar_b`).

### Fase 3: Progressão, Alternância e Conclusão com Revanche

Os turnos alternam estritamente entre as equipes. A partida segue até que uma das equipes atinja a pontuação máxima estipulada (ex.: 10 pontos). Ao atingir a meta, a fase muda para `FIM_JOGO`, exibindo a tela de vitória com métricas de sintonia da partida baseadas no histórico de `rodadas`. A sala oferece opções para disparar uma **Revanche** (mantendo equipes e zerando o placar) ou retornar ao lobby.

---

## 3. Arquitetura do Backend e Filosofia de Design

O backend e fundamentado no **PocketBase** estendido diretamente como framework em Go (`github.com/pocketbase/pocketbase`), gerando um unico binario executavel nativo com SQLite embutido via `modernc.org/sqlite` (`CGO_ENABLED=0`).

### Diretrizes Centrais Inegociáveis

1. **Consumo Minimo de Recursos:**
   - O processo deve operar entre **15 MB e 30 MB de RAM**, permitindo execucao continua em servidores modestos ou maquinas de desenvolvimento sem sobrecarga de heap.
2. **Anonimato e Ausencia de Contas:**
   - **NAO** crie tabelas de usuarios permanentes, senhas ou sistemas de OAuth.
   - Cada participante e identificado exclusivamente por um `player_id` (UUID efemero mantido no cliente).
3. **Efemeridade e Higienizacao Automatica:**
   - O banco SQLite nao retem partidas passadas.
   - O `CleanupComponent` e executado periodicamente a cada 5 minutos via agendador nativo do PocketBase, purgando jogadores inativos ha mais de 1 minuto e salas inativas ha mais de 5 minutos.
4. **Comunicação em Tempo Real Desacoplada:**
   - O front-end conecta-se via **Server-Sent Events (SSE)** nativo do PocketBase, escutando `salas`, `rodadas` e `jogadores`.
5. **Conectividade Externa via Tunel Reverso:**
   - O servidor escuta em `127.0.0.1:8090` e e exposto publicamente por **Tailscale Funnel** ou **Cloudflare Tunnel**, fornecendo terminacao TLS automatica.
6. **Composicao por Entidades e Componentes Atomicos:**
   - O backend e o arbitro autoritativo do sistema distribuido:
     - `EntidadeSessao` (`internal/entities/session.go`): salas e rodadas ativas.
     - `EntidadeParticipante` (`internal/entities/participant.go`): jogadores e equipes Corvo e Aguia.
     - `EntidadeDisputa` (`internal/entities/contest.go`): lances e resolucoes de confrontos.
     - `ProximityComponent` (`internal/components/proximity.go`): algoritmo matematico de pontuacao por tolerancia.
     - `SecretTargetComponent` (`internal/components/secret_target.go`): segredo seguro [5, 95] com `crypto/rand`.
     - `PptArbiterComponent` (`internal/components/ppt_arbiter.go`): arbitragem concorrente thread-safe (`sync.RWMutex`).
     - `CleanupComponent` (`internal/components/cleanup.go`): purga automatizada em segundo plano.
7. **Proibição de Commits e Pushes Autônomos:**
   - Agentes de IA são expressamente proibidos de executar `git commit`, `git tag` ou `git push` de forma autônoma.
   - O agente deve unicamente fornecer o sumário estruturado, a descrição detalhada e o comando git sugerido pronto para que o desenvolvedor humano realize a ação manualmente.

---

## 4. Estrutura do Repositorio e Papeis dos Arquivos

```text
backend/
├── cmd/
│   └── server/main.go               <-- Ponto de entrada do executavel PocketBase estendido
├── internal/
│   ├── entities/                    <-- EntidadeSessao, EntidadeParticipante e EntidadeDisputa
│   ├── components/                  <-- ProximityComponent, SecretTargetComponent, PptArbiter, Cleanup
│   │   └── components_test.go       <-- Testes unitarios Go com detector de condicoes de corrida
│   ├── hooks/                       <-- Registro de hooks autoritativos e rotas dedicadas
│   └── schema/                      <-- Auto-bootstrap de colecoes via pb_schema.json embarcado
├── scripts/
│   └── start_tunnel.sh              <-- Script utilitario de tunel reverso (Tailscale / Cloudflare)
├── .github/workflows/
│   └── release.yml                  <-- Pipeline de compilacao cruzada e releases no GitHub
├── CHANGELOG.md                     <-- Registro formal de versoes (Keep a Changelog)
├── CONTRACT.md                      <-- Especificacao de contratos REST, rotas e payloads
├── README.md                        <-- Instrucoes de compilacao, execucao e operacao
├── AGENTS.md                        <-- Este documento (fonte de verdade para agentes de IA)
├── pb_schema.json                   <-- Esquema declarativo de colecoes e regras
├── go.mod / go.sum                  <-- Modulos e dependencias de compilacao Go
├── .env.example                     <-- Modelo de variaveis de ambiente
└── .gitignore                       <-- Bloqueio de artefatos de build e banco local
```

---

## 5. Padroes de Desenvolvimento em Go Nativo

- **Zero CGO:** Toda compilacao deve manter `CGO_ENABLED=0`, garantindo portabilidade estatica absoluta entre Linux, macOS e Windows.
- **Concorrencia Segura:** Nunca compartilhe estado mutavel entre requisicoes sem sincronizacao nativa (`sync.RWMutex`, `sync.Map` ou canais).
- **Zero-Copy Pass-Through (SQL-Generated JSON):** Para consultas agregadas de alta performance (`/api/leveling/salas/{codigo}/snapshot`), utilize `json_object` e `json_group_array` diretamente no SQLite, repassando os bytes ao socket HTTP sem instanciar structs intermediarias ou invocar reflexao dinamica.
- **Auto-Bootstrap:** Colecoes essenciais devem ser garantidas no evento `OnBootstrap` a partir de `pb_schema.json` embarcado via `//go:embed`.

---

## 6. Comandos e Testes de Validacao

### Executar a suite de testes unitarios com verificacao de corrida

```bash
go test -v -race ./internal/components/...
```

### Compilar o executavel do servidor localmente

```bash
go build -o leveling-out-server ./cmd/server
```

### Iniciar o servidor

```bash
./leveling-out-server serve --http="127.0.0.1:8090"
```

### Testando a saude da API

```bash
curl -I http://127.0.0.1:8090/api/health
```

### Snapshot de alto desempenho via SQL-Generated JSON

```bash
curl -X GET "http://127.0.0.1:8090/api/leveling/salas/SALA_EXEMPLO/snapshot"
```
