# AGENTS.md: Diretrizes para Agentes Autônomos (Backend: Leveling Out)

Este documento é a fonte primária de verdade e instrução para qualquer desenvolvedor ou agente autônomo de inteligência artificial que opere neste repositório. Ele descreve o propósito do jogo **Leveling Out**, o fluxo de seu *game loop*, a arquitetura técnica adotada e os padrões estritos de desenvolvimento do backend.

---

## 1. Visão Geral do Jogo: Leveling Out

**Leveling Out** é um *party game* multiplayer cooperativo-competitivo focado em **dedução social, sintonia cultural e calibração conceitual**. Duas equipes competem tentando posicionar e adivinhar conceitos abstratos ao longo de uma escala contínua de 0% a 100%.

### Os 3 Pilares da Jogabilidade
1. **Nível Oculto:** O preenchimento visual de um tubo de ensaio com volume percentual aleatório (0% a 100%), visível **apenas** para o Codificador sorteado da equipe da vez.
2. **Gerar Pistas:** A partir de uma carta com um espectro bipolar (ex.: *"Famoso / Anônimo"*, *"Quente / Frio"*, *"Fácil / Difícil"*), o Codificador formula uma dica conceitual para orientar seu parceiro até o nível exato (ex.: meta de 70% em "Famoso/Anônimo" $\rightarrow$ pista: *"Keanu Reeves"*).
3. **Acerto Aproximado:** O Palpiteiro da equipe desloca um marcador analógico (*slider*) até onde estima estar o líquido. A pontuação é conferida por faixas de proximidade:
   - Margem de erro $\le 2\%$: **Na mosca (+4 pontos)**
   - Margem de erro $\le 6\%$: **Muito perto (+3 pontos)**
   - Margem de erro $\le 12\%$: **Perto (+2 pontos)**
   - Margem de erro $> 12\%$: **Errou (0 pontos)**

---

## 2. Game Loop Detalhado em 3 Fases

### Fase 1: Onboarding, Sala e Decisão de Turno
Os jogadores acessam a plataforma via navegador estático e entram em uma sala compartilhada via código de sala (`codigo`). Na sala, os participantes são distribuídos em duas equipes (**Equipe Azul / A** e **Equipe Vermelha / B**). Para definir quem começa, o sistema dispara um minijogo sincronizado de **Pedra, Papel e Tesoura** entre os capitães das equipes. A equipe vencedora ganha o direito de iniciar e seleciona o espectro conceitual da primeira rodada.

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

O backend é fundamentado no **PocketBase** (único binário executável em Go com SQLite embarcado) com suporte a extensões em JavaScript via runtime interno Goja (`pb_hooks/`).

### Diretrizes Centrais Inegociáveis:
1. **Consumo Mínimo de Recursos:**
   - O processo deve operar entre **15 MB e 30 MB de RAM**, permitindo execução contínua em máquinas modestas ou mini-PCs residenciais sem afetar outras tarefas do hospedeiro.
2. **Anonimato e Ausência de Contas:**
   - **NÃO** crie tabelas de usuários permanentes, sistemas de e-mail/senha ou OAuth.
   - Cada jogador é identificado apenas por um `player_id` (UUID efêmero gerado pelo cliente e armazenado no `sessionStorage`).
3. **Efemeridade e Zero Bloat (Higienização Automática):**
   - O banco de dados SQLite **não retém histórico de partidas passadas**.
   - O hook `pb_hooks/cleanup.pb.js` é executado a cada 5 minutos via agendador cron (`cronAdd`). Qualquer sala inativa há mais de **15 minutos** é purgada em cascata, deletando a sala, seus jogadores e o histórico de rodadas.
4. **Comunicação em Tempo Real Desacoplada:**
   - O front-end conecta-se exclusivamente via **Server-Sent Events (SSE)** nativo do PocketBase, escutando `salas` para o status global, `rodadas` para a jogada corrente e `jogadores` para presença.
   - O tráfego de saída do servidor deve ser mínimo e eficiente.
5. **Conectividade Externa via Túnel Reverso:**
   - O servidor escuta em `127.0.0.1:8090` e é exposto publicamente por **Tailscale Funnel** ou **Cloudflare Tunnel**, fornecendo terminação TLS (HTTPS/WSS) transparente mesmo sob redes móveis e CGNAT.

---

## 4. Estrutura do Repositório e Papéis dos Arquivos

```text
backend/
├── AGENTS.md                  <-- Este documento (regras e contexto para IA)
├── README.md                  <-- Instruções de operação humana e infraestrutura
├── CONTRACT.md                <-- Especificação formal de API REST, campos e SSE
├── pb_schema.json             <-- Esquema de dados exportado para importação rápida
├── pb_migrations/             <-- Migrações declarativas automatizadas em JS
│   └── 1790281200_create_game_collections.js <-- Criação de salas, rodadas e jogadores
├── pb_hooks/                  <-- Lógica em JavaScript do runtime Goja do PocketBase
│   ├── cleanup.pb.js          <-- Cron para purga de dados inativos (> 15 min) em cascata
│   └── game_rules.pb.js       <-- Validações de integridade, cálculo de pontos e sincronização
├── scripts/                   <-- Scripts utilitários de conveniência
│   ├── download_pocketbase.sh <-- Baixa binário oficial do PocketBase
│   └── start_tunnel.sh        <-- Exemplos de ativação de túnel (Tailscale/Cloudflare)
├── .env.example               <-- Exemplo de configuração de ambiente
└── .gitignore                 <-- Bloqueio de pb_data/ e binários locais
```

---

## 5. Convenções para Escrever Hooks no PocketBase (`pb_hooks/`)

O PocketBase executa JavaScript através do motor **Goja** (compatível com ES5/ES6). Fique atento aos padrões suportados:

### Regras de Sintaxe e Globais do Goja (PocketBase v0.23+ / v0.40+):
- Use as funções globais e métodos do `$app`:
  - `onRecordUpdateRequest((e) => { ... return e.next(); }, collectionName)`
  - `onRecordCreateRequest((e) => { ... return e.next(); }, collectionName)`
  - `cronAdd(tag, cronExpression, callback)`
  - `$app.findRecordsByFilter(collection, filter, sort, limit, offset, params)`
  - `$app.findRecordById(collection, id)`
  - `$app.delete(record)`
  - `$app.save(record)`
- Em todos os hooks de requisição (`*Request`), é obrigatório finalizar retornando `return e.next();` para que a requisição prossiga.
- **Autocontenção de callbacks:** Devido ao pool de instâncias concorrentes do Goja no PocketBase, declare funções auxiliares e regras diretamente no escopo de cada callback de hook para garantir que estejam disponíveis no contexto da requisição.
- **Não** utilize módulos Node.js (`require('fs')`, `npm`, `process.env`). Apenas funções nativas e APIs fornecidas pelo PocketBase (`$os`, `$http`, `$app`, etc.).
- Sempre valide tipos e garanta tratamento de erros com `try / catch` para evitar interrupção do serviço.

---

## 6. Comandos e Testes de Validação

### Testando a saúde da API:
```bash
curl -I http://127.0.0.1:8090/api/health
# Esperado: HTTP/1.1 200 OK
```

### Consultando salas ativas:
```bash
curl -X GET "http://127.0.0.1:8090/api/collections/salas/records"
```

### Simulando a purga de higienização manualmente:
Ao reiniciar o PocketBase ou aguardar o disparo do cron, acompanhe os logs no terminal para verificar as mensagens emitidas por `console.log("[CLEANUP] ...")`.
