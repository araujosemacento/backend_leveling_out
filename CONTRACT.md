# CONTRACT.md: Especificação de Dados e API Realtime

Este documento define o contrato de dados estrito e o protocolo de sincronização em tempo real entre o **Front-end (SvelteKit + p5.js)** e o **Backend (PocketBase)** para o jogo **Leveling Out**.

---

## 1. Coleções do PocketBase

A persistência do jogo é particionada em três coleções complementares com responsabilidades isoladas:

### 1.1. Coleção `salas`
Representa a instância de uma partida em andamento (ciclo macro de sessão e placar acumulado).

| Campo | Tipo | Obrigatório | Descrição / Valores Permitidos |
| :--- | :--- | :---: | :--- |
| `id` | `TEXT` (15 chars) | Sim | Identificador único autogerado pelo PocketBase. |
| `codigo` | `TEXT` | Sim | Código alfanumérico da sala (ex.: `"alpha-42"`). Índice único. |
| `fase` | `SELECT` | Sim | Estado global da partida: `LOBBY`, `INICIATIVA`, `EM_RODADA`, `FIM_JOGO`. |
| `rodada_atual` | `NUMBER` | Sim | Número ordinal da rodada em andamento (inicia em `1`). |
| `equipe_ativa` | `SELECT` | Sim | Equipe que joga no turno atual: `'A'` ou `'B'`. |
| `placar_a` | `NUMBER` | Não | Pontuação acumulada da Equipe A (default: `0`). |
| `placar_b` | `NUMBER` | Não | Pontuação acumulada da Equipe B (default: `0`). |
| `vencedor` | `SELECT` | Não | Equipe campeã da partida: `''`, `'A'` ou `'B'`. |
| `created` | `AUTODATE` | Sim | Timestamp UTC de criação do registro. |
| `updated` | `AUTODATE` | Sim | Timestamp UTC da última atualização. |

#### Regras de Acesso da Coleção `salas`:
1. **List / View:** `""` (público / anônimo permitido)
2. **Create:** `""` (qualquer jogador pode inicializar uma sala)
3. **Update:** `""` (atualizações autorizadas por sala, validadas por hooks)
4. **Delete:** `null` (bloqueado para clientes públicos; apenas hooks internos e admins podem deletar)

---

### 1.2. Coleção `rodadas`
Registra cada jogada individual da partida (ciclo micro de dedução e histórico para métricas de sintonia).

| Campo | Tipo | Obrigatório | Descrição / Valores Permitidos |
| :--- | :--- | :---: | :--- |
| `id` | `TEXT` (15 chars) | Sim | Identificador interno no PocketBase. |
| `sala_codigo` | `TEXT` | Sim | Código da sala vinculada. |
| `numero` | `NUMBER` | Sim | Índice ordinal da rodada (1, 2, 3...). |
| `equipe` | `SELECT` | Sim | Equipe ativa na rodada: `'A'` ou `'B'`. |
| `fase_rodada` | `SELECT` | Sim | Fase do turno: `ESCOLHA_ESPECTRO`, `DICA`, `PALPITE`, `REVELADA`, `CONCLUIDA`. |
| `espectro_esquerda` | `TEXT` | Não | Polo esquerdo da carta sorteada (ex.: `"Famoso"`). |
| `espectro_direita` | `TEXT` | Não | Polo direito da carta sorteada (ex.: `"Anônimo"`). |
| `meta_oculta` | `NUMBER` | Não | Nível percentual secreto sorteado pelo hook (5 a 95). |
| `dica` | `TEXT` | Não | Pista textual cadastrada pelo Codificador. |
| `palpite` | `NUMBER` | Não | Valor percentual calibrado pelo Palpiteiro (0 a 100). |
| `pontos` | `NUMBER` | Não | Pontos obtidos na jogada (`4`, `3`, `2` ou `0`). |
| `created` | `AUTODATE` | Sim | Timestamp UTC de criação do registro. |
| `updated` | `AUTODATE` | Sim | Timestamp UTC da última atualização. |

#### Regras de Acesso da Coleção `rodadas`:
1. **List / View:** `""` (público)
2. **Create:** `""` (público)
3. **Update:** `""` (público, validado por hooks Goja)
4. **Delete:** `""` (permitido para manutenção)

---

### 1.3. Coleção `jogadores`
Representa os participantes presentes em uma sala e o estado de presença.

| Campo | Tipo | Obrigatório | Descrição / Valores Permitidos |
| :--- | :--- | :---: | :--- |
| `id` | `TEXT` (15 chars) | Sim | Identificador interno no PocketBase. |
| `sala_codigo` | `TEXT` | Sim | Código da sala à qual o jogador pertence. |
| `player_id` | `TEXT` | Sim | UUID gerado pelo navegador cliente e salvo em `sessionStorage`. |
| `nome` | `TEXT` | Sim | Nome de exibição do jogador (ex.: `"Jogador 1042"`). |
| `equipe` | `SELECT` | Sim | Equipe vinculada: `'A'`, `'B'` ou `'ESPECTADOR'`. |
| `papel` | `SELECT` | Sim | Papel no turno: `'CODIFICADOR'`, `'PALPITEIRO'` ou `'ESPECTADOR'`. |
| `last_seen` | `DATE` | Sim | Timestamp UTC atualizado via heartbeat para detecção de presença. |
| `created` | `AUTODATE` | Sim | Timestamp UTC de criação do registro. |
| `updated` | `AUTODATE` | Sim | Timestamp UTC da última atualização. |

#### Regras de Acesso da Coleção `jogadores`:
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

$$\text{pontos} = \begin{cases} 
4, & \text{se } \Delta \le 2\% \quad \text{(Na mosca)} \\
3, & \text{se } 3\% \le \Delta \le 6\% \quad \text{(Muito perto)} \\
2, & \text{se } 7\% \le \Delta \le 12\% \quad \text{(Perto)} \\
0, & \text{se } \Delta > 12\% \quad \text{(Fora da margem)}
\end{cases}$$

O hook em `game_rules.pb.js` grava o campo `pontos` no registro de `rodadas` e incrementa automaticamente o placar da equipe ativa (`placar_a` ou `placar_b`) na coleção `salas`. Caso a pontuação acumulada atinja o limite estipulado ($\ge 10$ pontos), a sala é automaticamente declarada com o vencedor e avança para `FIM_JOGO`.
