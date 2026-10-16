# CONTRACT.md: Especificação de Dados e API Realtime

Este documento define o contrato de dados estrito e o protocolo de sincronização em tempo real entre o **Front-end (SvelteKit + p5.js)** e o **Backend (PocketBase)** para o jogo **Leveling Out**.

---

## 1. Coleções do PocketBase

A persistência do jogo é particionada em três coleções complementares com responsabilidades isoladas:

### 1.1. Coleção `salas`

Representa a instância de uma partida em andamento (ciclo macro de sessão e placar acumulado).

| Campo          | Tipo              | Obrigatório | Descrição / Valores Permitidos                                            |
| :------------- | :---------------- | :---------: | :------------------------------------------------------------------------ |
| `id`           | `TEXT` (15 chars) |     Sim     | Identificador único autogerado pelo PocketBase.                           |
| `codigo`       | `TEXT`            |     Sim     | Código alfanumérico da sala (ex.: `"alpha-42"`). Índice único.            |
| `fase`         | `SELECT`          |     Sim     | Estado global da partida: `LOBBY`, `INICIATIVA`, `EM_RODADA`, `FIM_JOGO`. |
| `rodada_atual` | `NUMBER`          |     Sim     | Número ordinal da rodada em andamento (inicia em `1`).                    |
| `equipe_ativa` | `SELECT`          |     Sim     | Equipe que joga no turno atual: `'A'` ou `'B'`.                           |
| `placar_a`     | `NUMBER`          |     Não     | Pontuação acumulada da Equipe A (default: `0`).                           |
| `placar_b`     | `NUMBER`          |     Não     | Pontuação acumulada da Equipe B (default: `0`).                           |
| `vencedor`     | `SELECT`          |     Não     | Equipe campeã da partida: `''`, `'A'` ou `'B'`.                           |
| `host_id`      | `TEXT`            |     Não     | UUID do jogador anfitrião e dono da sala.                                 |
| `created`      | `AUTODATE`        |     Sim     | Timestamp UTC de criação do registro.                                     |
| `updated`      | `AUTODATE`        |     Sim     | Timestamp UTC da última atualização.                                      |

#### Regras de Acesso da Coleção `salas`

1. **List / View:** `""` (público / anônimo permitido)
2. **Create:** `""` (qualquer jogador pode inicializar uma sala)
3. **Update:** `""` (atualizações autorizadas por sala, validadas por hooks)
4. **Delete:** `null` (bloqueado para clientes públicos; apenas hooks internos e admins podem deletar)

---

### 1.2. Coleção `rodadas`

Registra cada jogada individual da partida (ciclo micro de dedução e histórico para métricas de sintonia).

| Campo               | Tipo              | Obrigatório | Descrição / Valores Permitidos                                                 |
| :------------------ | :---------------- | :---------: | :----------------------------------------------------------------------------- |
| `id`                | `TEXT` (15 chars) |     Sim     | Identificador interno no PocketBase.                                           |
| `sala_codigo`       | `TEXT`            |     Sim     | Código da sala vinculada.                                                      |
| `numero`            | `NUMBER`          |     Sim     | Índice ordinal da rodada (1, 2, 3...).                                         |
| `equipe`            | `SELECT`          |     Sim     | Equipe ativa na rodada: `'A'` ou `'B'`.                                        |
| `fase_rodada`       | `SELECT`          |     Sim     | Fase do turno: `ESCOLHA_ESPECTRO`, `DICA`, `PALPITE`, `REVELADA`, `CONCLUIDA`. |
| `espectro_esquerda` | `TEXT`            |     Não     | Polo esquerdo da carta sorteada (ex.: `"Famoso"`).                             |
| `espectro_direita`  | `TEXT`            |     Não     | Polo direito da carta sorteada (ex.: `"Anônimo"`).                             |
| `meta_oculta`       | `NUMBER`          |     Não     | Nível percentual secreto sorteado pelo hook (5 a 95).                          |
| `dica`              | `TEXT`            |     Não     | Pista textual cadastrada pelo Codificador.                                     |
| `palpite`           | `NUMBER`          |     Não     | Valor percentual calibrado pelo Palpiteiro (0 a 100).                          |
| `pontos`            | `NUMBER`          |     Não     | Pontos obtidos na jogada (`4`, `3`, `2` ou `0`).                               |
| `created`           | `AUTODATE`        |     Sim     | Timestamp UTC de criação do registro.                                          |
| `updated`           | `AUTODATE`        |     Sim     | Timestamp UTC da última atualização.                                           |

#### Regras de Acesso da Coleção `rodadas`

1. **List / View:** `""` (público)
2. **Create:** `""` (público)
3. **Update:** `""` (público, validado por hooks Goja)
4. **Delete:** `""` (permitido para manutenção)

---

### 1.3. Coleção `jogadores`

Representa os participantes presentes em uma sala e o estado de presença.

| Campo         | Tipo              | Obrigatório | Descrição / Valores Permitidos                                     |
| :------------ | :---------------- | :---------: | :----------------------------------------------------------------- |
| `id`          | `TEXT` (15 chars) |     Sim     | Identificador interno no PocketBase.                               |
| `sala_codigo` | `TEXT`            |     Sim     | Código da sala à qual o jogador pertence.                          |
| `player_id`   | `TEXT`            |     Sim     | UUID gerado pelo navegador cliente e salvo em `sessionStorage`.    |
| `nome`        | `TEXT`            |     Sim     | Nome de exibição do jogador (ex.: `"Jogador 1042"`).               |
| `equipe`      | `SELECT`          |     Sim     | Equipe vinculada: `'A'`, `'B'`, `'ALEATORIO'` ou `'ESPECTADOR'`.   |
| `papel`       | `SELECT`          |     Sim     | Papel no turno: `'CODIFICADOR'`, `'PALPITEIRO'` ou `'ESPECTADOR'`. |
| `is_host`     | `BOOL`            |     Não     | Define se o jogador é o anfitrião e dono da sala (default: false). |
| `last_seen`   | `DATE`            |     Sim     | Timestamp UTC atualizado via heartbeat para detecção de presença.  |
| `created`     | `AUTODATE`        |     Sim     | Timestamp UTC de criação do registro.                              |
| `updated`     | `AUTODATE`        |     Sim     | Timestamp UTC da última atualização.                               |

#### Regras de Acesso da Coleção `jogadores`

1. **List / View:** `""` (público)
2. **Create:** `""` (público)
3. **Update:** `""` (público, restrito a atualizar seu próprio `last_seen` ou dados locais)
4. **Delete:** `""` (permite saída voluntária da sala)

---

## 2. Protocolo de Sincronização em Tempo Real (SSE)

O cliente web conecta-se usando o SDK oficial do PocketBase:

```javascript
import PocketBase from 'pocketbase';

const pb = new PocketBase(SERVER_PUBLIC_URL);

// 1. Assinatura da Sala (Ciclo Macro e Placar)
pb.collection('salas').subscribe(salaRecordId, (e) => {
 if (e.action === 'update') {
  gameEngine.syncSala(e.record);
 }
});

// 2. Assinatura da Rodada Ativa (Tubo, Dica, Palpite e Revelação)
pb.collection('rodadas').subscribe('*', (e) => {
 if (e.record.sala_codigo === currentRoomCode) {
  gameEngine.syncRodada(e.action, e.record);
 }
});

// 3. Assinatura de Presença dos Jogadores
pb.collection('jogadores').subscribe('*', (e) => {
 if (e.record.sala_codigo === currentRoomCode) {
  lobbyEngine.syncJogadores(e.action, e.record);
 }
});
```

---

## 3. Ciclo de Heartbeat e Presença

1. A cada **10 segundos**, o cliente envia um `PATCH` para o seu próprio registro em `jogadores`:

   ```javascript
   pb.collection('jogadores').update(meuJogadorDbId, {
    last_seen: new Date().toISOString()
   });
   ```

2. Um jogador sem atualização de `last_seen` há mais de **30 segundos** é considerado desconectado na interface do jogo.

---

## 4. Regra de Pontuação por Proximidade

Quando a fase da rodada transita para `REVELADA`, o backend calcula a distância absoluta:

$$\Delta = |\text{meta\_oculta} - \text{palpite}|$$

$$ \text{pontos} = \begin{cases}
4, & \text{se } \Delta \le 2\% \quad \text{(Na mosca)} \\
3, & \text{se } 3\% \le \Delta \le 6\% \quad \text{(Muito perto)} \\
2, & \text{se } 7\% \le \Delta \le 12\% \quad \text{(Perto)} \\
0, & \text{se } \Delta > 12\% \quad \text{(Fora da margem)}
\end{cases}$$

O hook autoritativo grava o campo `pontos` no registro de `rodadas` e incrementa automaticamente o placar da equipe ativa (`placar_a` ou `placar_b`) na coleção `salas`. Caso a pontuação acumulada atinja o limite estipulado ($\ge 10$ pontos), a sala é automaticamente declarada com o vencedor e avança para `FIM_JOGO`.

---

## 5. Endpoints Autoritativos Dedicados

### 5.1. Iniciar Partida e Balanceamento Estocástico

- **Rota:** `POST /api/leveling/salas/{codigo}/iniciar`
- **Permissão:** Restrito ao anfitrião da sala (`host_id`).
- **Payload:**
  ```json
  {
    "player_id": "uuid-do-host"
  }
  ```
- **Comportamento Autoritativo:**
  1. Verifica se a sala está com quórum completo (exatamente 4 jogadores ativos).
  2. Identifica os jogadores com `equipe = 'ALEATORIO'`.
  3. Sorteia estocasticamente (`crypto/rand`) esses jogadores entre as vagas remanescentes das equipes Pato (`A`) e Ganso (`B`) até totalizar exatamente 2 integrantes em cada.
  4. Transiciona o estado da sala para `fase = "INICIATIVA"`.
  5. Dispara evento SSE notificando todos os clientes conectados.
- **Respostas HTTP:**
  - `200 OK`: `{ "message": "Partida iniciada com sucesso", "fase": "INICIATIVA" }`
  - `400 Bad Request`: Sala incompleta (`"A partida requer exatamente 4 jogadores para iniciar"`) ou permissão negada (`"Apenas o dono da sala pode iniciar a partida"`).

### 5.2. Expulsão de Participante pelo Anfitrião (Kick de Jogador)

- **Rota:** `POST /api/leveling/salas/{codigo}/expulsar`
- **Permissão:** Restrito ao anfitrião da sala (`host_id`).
- **Headers:** `Content-Type: application/json`
- **Payload:**
  ```json
  {
    "host_player_id": "uuid-do-host",
    "alvo_player_id": "uuid-do-participante-alvo"
  }
  ```
- **Comportamento Autoritativo:**
  1. Localiza a sala pelo `codigo` informado na rota e valida sua existência.
  2. Garante que a sala se encontra estritamente na fase `LOBBY` (a expulsão é bloqueada durante partidas em andamento).
  3. Valida se o `host_player_id` coincide de forma soberana com o `host_id` registrado na sala (Zero Trust).
  4. Impede auto-expulsão: se `alvo_player_id == host_player_id`, a requisição é rejeitada (o anfitrião deve utilizar a rota de cancelamento ou saída voluntária).
  5. Localiza o registro do participante em `jogadores` correspondente a `alvo_player_id` vinculado à sala.
  6. Remove o registro do participante do banco SQLite/PocketBase.
  7. Se o jogador estava alocado em uma equipe (`A`, `B` ou `ALEATORIO`), a respectiva vaga é liberada imediatamente.
  8. Dispara evento SSE (`action = "delete"` na coleção `jogadores`) para todos os clientes conectados sincronizarem a composição de equipes e recalcularem o quórum em tempo real.
  9. O cliente do jogador expulso identifica a perda autoritativa de assento ou recebe notificação de desconexão forçada, sendo redirecionado para a tela inicial (`/`) com aviso bloqueante de expulsão pelo anfitrião.

### 5.3. Disputa de Iniciativa (Pedra, Papel e Tesoura)

- **Rota:** `POST /api/leveling/salas/{codigo}/ppt/lance`
- **Permissão:** Restrito ao representante da equipe na disputa.
- **Headers:**
  - `Content-Type: application/json`
  - `X-Player-Id`: ID do jogador representante
  - `X-Session-Token`: Token criptográfico de sessão
- **Payload:**
  ```json
  {
    "lance": "pedra" | "papel" | "tesoura"
  }
  ```
- **Comportamento Autoritativo:**
  1. Identifica a equipe do jogador e valida se ele é o representante oficial (menor `equipe_desde` na equipe).
  2. Garante que a disputa está na fase `INICIATIVA` e subfase `ESCOLHA` dentro do prazo de 10 segundos.
  3. Lance definitivo: impede alteração de lance já submetido.
  4. Mantém o sigilo do lance em memória (`PptArbiterComponent`) até que ambas as equipes tenham jogado (ou até o timeout/preenchimento).
  5. Se ambos os lances forem recebidos, inicia imediatamente a contagem regressiva de revelação (`ppt_estado = "CONTAGEM"`).
  6. Em caso de timeout (10s) ou desconexão do representante, o servidor realiza um lance aleatório por ele.
  7. Após o "Já!", arbitra o resultado: empate reabre a disputa; vitória define `equipe_ativa` e transiciona automaticamente para `ESCOLHA_ESPECTRO`.

### 5.4. Consulta de Lance Sigiloso da Própria Equipe

- **Rota:** `GET /api/leveling/salas/{codigo}/ppt/meu-lance`
- **Permissão:** Participante autenticado da sala pertencente a uma equipe.
- **Headers:**
  - `X-Player-Id`: ID do jogador
  - `X-Session-Token`: Token criptográfico de sessão
- **Comportamento Autoritativo:**
  - Retorna o lance escolhido pelo representante da equipe do solicitante, permitindo que os espectadores da mesma equipe visualizem `"Sua equipe escolheu [lance]"` sem quebrar o sigilo perante a equipe adversária.
- **Respostas HTTP:**
  - `200 OK`: `{ "lance": "pedra" | "papel" | "tesoura" | "" }`
  - `401 Unauthorized`: Sessão inválida ou expirada.
  - `404 Not Found`: Sala ou jogador não encontrado.
