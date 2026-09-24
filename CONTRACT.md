# CONTRACT.md: Especificação de Dados e API Realtime

Este documento define o contrato de dados estrito e o protocolo de sincronização em tempo real entre o **Front-end (SvelteKit + p5.js)** e o **Backend (PocketBase)** para o jogo **Leveling Out**.

---

## 1. Coleções do PocketBase

### 1.1. Coleção `salas`
Representa a instância de uma partida em andamento.

| Campo | Tipo | Obrigatório | Descrição / Valores Permitidos |
| :--- | :--- | :---: | :--- |
| `id` | `TEXT` (15 chars) | Sim | Identificador único autogerado pelo PocketBase. |
| `codigo` | `TEXT` | Sim | Código alfanumérico da sala (ex.: `"alpha-42"`). Índice único. |
| `fase` | `SELECT` | Sim | Estado do game loop:<br>• `LOBBY`<br>• `INICIATIVA` (minijogo Pedra-Papel-Tesoura)<br>• `ESCOLHA_ESPECTRO`<br>• `RODADA_DICA`<br>• `RODADA_PALPITE`<br>• `REVELACAO`<br>• `FIM_JOGO` |
| `rodada` | `NUMBER` | Sim | Número ordinal da rodada (inicia em `1`). |
| `equipe_ativa` | `SELECT` | Sim | Equipe que joga no turno atual: `'A'` (Azul) ou `'B'` (Vermelha). |
| `espectro_esquerda` | `TEXT` | Não | Extremo esquerdo da carta conceitual (ex.: `"Famoso"`). |
| `espectro_direita` | `TEXT` | Não | Extremo direito da carta conceitual (ex.: `"Anônimo"`). |
| `meta_oculta` | `NUMBER` | Não | Nível percentual real (inteiro entre `0` e `100`). |
| `dica` | `TEXT` | Não | Pista textual formulada pelo Codificador da vez. |
| `palpite` | `NUMBER` | Não | Palpite percentual formulado pelo Palpiteiro (inteiro entre `0` e `100`). |
| `pontos_rodada` | `NUMBER` | Não | Pontuação obtida na revelação (`0`, `2`, `3` ou `4`). |
| `placar_a` | `NUMBER` | Sim | Pontuação acumulada da Equipe Azul (default: `0`). |
| `placar_b` | `NUMBER` | Sim | Pontuação acumulada da Equipe Vermelha (default: `0`). |
| `vencedor` | `SELECT` | Não | Equipe campeã da partida: `''`, `'A'` ou `'B'`. |

#### Regras de Acesso da Coleção `salas`:
- **List / View:** `""` (público / anônimo permitido)
- **Create:** `""` (qualquer jogador pode inicializar uma sala)
- **Update:** `""` (atualizações autorizadas por sala, validadas por hooks)
- **Delete:** `@request.auth.id != ""` (bloqueado para clientes públicos; apenas hooks internos e admins podem deletar)

---

### 1.2. Coleção `jogadores`
Representa os participantes presentes em uma sala.

| Campo | Tipo | Obrigatório | Descrição / Valores Permitidos |
| :--- | :--- | :---: | :--- |
| `id` | `TEXT` (15 chars) | Sim | Identificador interno no PocketBase. |
| `sala_codigo` | `TEXT` | Sim | Código da sala à qual o jogador pertence (para filtros rápidos). |
| `player_id` | `TEXT` | Sim | UUID gerado pelo navegador cliente e salvo em `sessionStorage`. |
| `nome` | `TEXT` | Sim | Nome de exibição do jogador (ex.: `"Jogador #1042"`). |
| `equipe` | `SELECT` | Sim | Equipe do jogador: `'A'`, `'B'` ou `'ESPECTADOR'`. |
| `papel` | `SELECT` | Sim | Papel no turno: `'CODIFICADOR'`, `'PALPITEIRO'` ou `'ESPECTADOR'`. |
| `last_seen` | `DATE` | Sim | Timestamp UTC atualizado via heartbeat para detecção de presença. |

#### Regras de Acesso da Coleção `jogadores`:
- **List / View:** `""` (público)
- **Create:** `""` (público)
- **Update:** `""` (público, restrito a atualizar seu próprio `last_seen` ou dados locais)
- **Delete:** `""` (permite saída voluntária da sala)

---

## 2. Protocolo de Sincronização em Tempo Real (SSE)

O cliente web conecta-se usando a biblioteca JavaScript oficial do PocketBase (`pocketbase.es.mjs`):

```javascript
import PocketBase from 'pocketbase';

const pb = new PocketBase(SERVER_PUBLIC_URL);

// Assinatura de mudanças na sala em tempo real
pb.collection('salas').subscribe(recordId, (e) => {
    if (e.action === 'update') {
        const salaAtualizada = e.record;
        // Sincroniza a máquina de estados no p5.js
        gameEngine.syncFromBackend(salaAtualizada);
    }
});

// Assinatura da lista de jogadores da sala
pb.collection('jogadores').subscribe('*', (e) => {
    if (e.record.sala_codigo === currentRoomCode) {
        // Atualiza contagem de presença e listas de equipe
        updateLobbyPresence(e.action, e.record);
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
2. Um jogador sem atualização de `last_seen` há mais de **30 segundos** é considerado desconectado na interface do p5.js.

---

## 4. Regra de Pontuação por Proximidade

Quando a fase transita para `REVELACAO`, a pontuação da rodada é calculada estritamente com base na diferença absoluta:

$$\Delta = |\text{meta\_oculta} - \text{palpite}|$$

$$\text{pontos} = \begin{cases} 
4, & \text{se } \Delta \le 2\% \quad \text{(Na mosca!)} \\
3, & \text{se } 3\% \le \Delta \le 6\% \quad \text{(Muito perto)} \\
2, & \text{se } 7\% \le \Delta \le 12\% \quad \text{(Perto)} \\
0, & \text{se } \Delta > 12\% \quad \text{(Fora da margem)}
\end{cases}$$

Os pontos calculados são somados ao placar da equipe ativa (`placar_a` ou `placar_b`). Se o placar atingir a pontuação limite ($\ge 10$), a sala transita para `FIM_JOGO`.
