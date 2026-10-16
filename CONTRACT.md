# CONTRACT.md: Especificação de Protocolo WebSocket e API Realtime

Este documento define formalmente o contrato de dados estrito e o protocolo de comunicação bidirecional em tempo real entre o **Front-end (SvelteKit)** e o **Servidor Autoritário Go em Memória** para o jogo **Leveling Out**.

---

## 1. Topologia de Conexão: WebSocket Full-Duplex

O jogo opera com um modelo de comunicação bidirecional de baixa latência baseado no protocolo WebSocket (RFC 6455). Não há persistência relacional em disco nem banco de dados intermediário. Cada sala é um **Ator em Memória** (`RoomActor`) executando em uma goroutine Go dedicada.

### 1.1. Endpoint de Handshake e Autenticação

- **URL:** `ws://<servidor>/ws/salas/{codigo}` (ou `wss://` em produção com TLS)
- **Método HTTP:** `GET` com cabeçalhos de Upgrade para WebSocket (`Upgrade: websocket`, `Connection: Upgrade`)
- **Query Parameters de Autenticação:**

| Parâmetro      | Tipo     | Obrigatório | Descrição                                                                                             |
| :------------- | :------- | :---------: | :---------------------------------------------------------------------------------------------------- |
| `playerId`     | `string` |     Sim     | UUID único persistido no armazenamento local (`sessionStorage`) do participante.                      |
| `sessionToken` | `string` |     Não     | Token criptográfico emitido pelo servidor no primeiro handshake. Usado para reconectar na mesma vaga. |
| `nome`         | `string` |     Sim     | Apelido do jogador (sanitizado contra HTML, tamanho de 1 a 20 caracteres).                            |
| `secret`       | `string` |     Não     | Segredo compartilhado de infraestrutura (`PROXY_SECRET`) para validar requisições via proxy reverso.  |

### 1.2. Ciclo de Conexão

1. O cliente abre a conexão WebSocket enviando os parâmetros de handshake.
2. O servidor valida os parâmetros e o proxy secret (se configurado).
3. Se a sala não existir, o servidor cria a sala em memória e define este jogador como anfitrião (`isHost = true`).
4. Se a sala existir, o jogador é associado à sua sessão anterior (via `sessionToken`) ou registrado em nova vaga (quórum máximo de 4 jogadores).
5. O servidor responde imediatamente com o envelope `BEM_VINDO`, contendo o `sessionToken` oficial e o estado projetado da sala.
6. A partir deste momento, a própria conexão TCP/WebSocket estabelece a identidade do jogador, dispensando envio repetido de tokens nos comandos.

---

## 2. Envelopes de Entrada (Comandos do Cliente)

Todas as mensagens enviadas do cliente para o servidor são objetos JSON com a propriedade discriminadora `tipo`.

### 2.1. `ESCOLHER_EQUIPE`

Define a equipe do jogador durante o Lobby.

```json
{
  "tipo": "ESCOLHER_EQUIPE",
  "equipe": "A" | "B" | "ALEATORIO"
}
```

- **Restrições:** Cada equipe (`A` ou `B`) suporta no máximo 2 jogadores. A opção `ALEATORIO` acolhe jogadores indecisos.

### 2.2. `INICIAR_PARTIDA`

Inicia a partida, sorteia participantes indecisos e transiciona para a disputa de iniciativa.

```json
{
	"tipo": "INICIAR_PARTIDA"
}
```

- **Restrições:** Exclusivo do anfitrião (`host_id`). Requer quórum exato de 4 participantes ativos na sala.

### 2.3. `EXPULSAR_JOGADOR`

Remove autoritativamente um participante da sala.

```json
{
	"tipo": "EXPULSAR_JOGADOR",
	"alvoId": "uuid-do-jogador-alvo"
}
```

- **Restrições:** Exclusivo do anfitrião (`host_id`). Permitido apenas durante a fase `LOBBY`. Impede auto-expulsão. O participante alvo recebe o envelope `KICKADO` e tem seu socket desconectado.

### 2.4. `DESFAZER_PARTIDA`

Cancela a partida em andamento e retorna todos os participantes para a fase `LOBBY`.

```json
{
	"tipo": "DESFAZER_PARTIDA"
}
```

- **Restrições:** Exclusivo do anfitrião (`host_id`).

### 2.5. `LANCE_PPT`

Submete a jogada do representante da equipe no minijogo de Pedra, Papel e Tesoura.

```json
{
  "tipo": "LANCE_PPT",
  "lance": "pedra" | "papel" | "tesoura"
}
```

- **Restrições:** Permitido apenas para o representante da equipe (jogador com menor timestamp de ingresso na equipe) durante a subfase `ESCOLHA` de `INICIATIVA`. O lance é mantido em sigilo até que ambas as equipes tenham jogado ou ocorra timeout de 10s.

### 2.6. `ESCOLHER_ESPECTRO`

Seleciona a carta de espectro bipolar para a rodada.

```json
{
	"tipo": "ESCOLHER_ESPECTRO",
	"cartaId": "string-identificador-da-carta"
}
```

- **Restrições:** Permitido apenas para o Codificador da equipe vencedora da iniciativa (ou da equipe da vez).

### 2.7. `SUBMETER_DICA`

Registra a pista conceitual elaborada pelo Codificador a partir da meta secreta observada no tubo.

```json
{
	"tipo": "SUBMETER_DICA",
	"dica": "Texto da pista (1 a 60 caracteres)"
}
```

- **Restrições:** Permitido apenas para o Codificador da rodada durante a fase `RODADA_DICA`. A pista é sanitizada contra injeção e quebras de linha.

### 2.8. `SLIDER_MOVER`

Transmite a posição analógica do marcador em tempo real para os membros da sala.

```json
{
	"tipo": "SLIDER_MOVER",
	"percentual": 42.5
}
```

- **Restrições:** Permitido apenas para o Palpiteiro da rodada durante a fase `RODADA_PALPITE`. O servidor retransmite essa posição via `SLIDER_STREAM` para todos os demais membros da sala.

### 2.9. `CONFIRMAR_PALPITE`

Fixa o palpite definitivo da equipe e dispara a revelação dramática de pontuação.

```json
{
	"tipo": "CONFIRMAR_PALPITE",
	"palpite": 42.5
}
```

- **Restrições:** Permitido apenas para o Palpiteiro da rodada durante a fase `RODADA_PALPITE`. O valor sofre clamp matemático entre 0 e 100.

### 2.10. `REVANCHE`

Dispara o reinício da partida mantendo a sala e os jogadores.

```json
{
	"tipo": "REVANCHE"
}
```

- **Restrições:** Permitido durante a fase `VITORIA`. Zera o placar e reinicia a partir da disputa de iniciativa (PPT).

---

## 3. Envelopes de Saída (Servidor para Cliente)

### 3.1. `BEM_VINDO`

Enviado imediatamente após a aceitação do handshake WebSocket.

```json
{
  "tipo": "BEM_VINDO",
  "playerId": "uuid-do-jogador",
  "sessionToken": "token-criptografico-de-sessao",
  "salaCodigo": "codigo-da-sala",
  "isHost": true,
  "sala": { ... }
}
```

### 3.2. `SYNC`

Broadcast disparado sempre que ocorre qualquer mutação autoritativa de estado na sala.

```json
{
  "tipo": "SYNC",
  "sala": {
    "codigo": "sala-123",
    "fase": "LOBBY" | "PPT_DISPUTA" | "RODADA_ESCOLHA_ESPECTRO" | "RODADA_DICA" | "RODADA_PALPITE" | "RODADA_REVELACAO" | "VITORIA",
    "placarA": 4,
    "placarB": 2,
    "equipeAtiva": "A",
    "hostId": "uuid-do-host",
    "vencedor": "" | "A" | "B",
    "rodadaAtual": 2
  },
  "rodadaAtiva": {
    "numero": 2,
    "equipe": "A",
    "fase": "RODADA_PALPITE",
    "codificadorId": "uuid-do-codificador",
    "palpiteiroId": "uuid-do-palpiteiro",
    "espectro": {
      "id": "quente_frio",
      "poloEsquerdo": "Quente",
      "poloDireito": "Frio"
    },
    "espectrosDisponiveis": [ ... ],
    "metaOculta": null,
    "dica": "Cafe fervendo",
    "palpite": 78,
    "pontos": 0
  },
  "historicoRodadas": [ ... ],
  "ppt": {
    "fase": "AGUARDANDO" | "ESCOLHA" | "CONTAGEM" | "RESULTADO",
    "repAId": "uuid-do-representante-a",
    "repBId": "uuid-do-representante-b",
    "lanceA": "pedra",
    "lanceB": "tesoura",
    "lanceRecebidoA": true,
    "lanceRecebidoB": true,
    "tempoRestante": 10,
    "contagemRegressiva": 3,
    "vencedor": "A"
  },
  "jogadores": [
    {
      "id": "uuid-do-jogador",
      "nome": "Apelido",
      "equipe": "A",
      "papel": "PALPITEIRO",
      "isHost": true,
      "isOnline": true
    }
  ]
}
```

> **Princípio Zero Trust no Despacho:** O campo `metaOculta` é omitido (`null`) para todos os jogadores durante as fases de escolha, dica e palpite, sendo enviado unicamente para o Codificador oficial da vez na fase de Dica. O segredo só é tornado público para todos na fase `RODADA_REVELACAO` ou `VITORIA`.

### 3.3. `SLIDER_STREAM`

Transmitido em tempo real para os membros da sala enquanto o palpiteiro arrasta o slider.

```json
{
	"tipo": "SLIDER_STREAM",
	"percentual": 64.2,
	"autorId": "uuid-do-palpiteiro"
}
```

### 3.4. `KICKADO`

Enviado unicamente para a conexão WebSocket do participante expulso antes do encerramento forçado do canal.

```json
{
	"tipo": "KICKADO",
	"motivo": "Voce foi expulso da sala pelo anfitriao."
}
```

### 3.5. `ERRO`

Enviado em caso de comando inválido, violação de regras ou falha de autorização.

```json
{
	"tipo": "ERRO",
	"mensagem": "Descricao do erro amigavel"
}
```

---

## 4. Regras Matemáticas de Pontuação por Proximidade

Quando o palpite é confirmado, a pontuação é calculada de forma soberana pelo `ProximityComponent`:

$$\Delta = |\text{meta\_oculta} - \text{palpite}|$$

$$ \text{pontos} = \begin{cases}
4, & \text{se } \Delta \le 2\% \quad \text{(Na mosca)} \\
3, & \text{se } 3\% \le \Delta \le 6\% \quad \text{(Muito perto)} \\
2, & \text{se } 7\% \le \Delta \le 12\% \quad \text{(Perto)} \\
0, & \text{se } \Delta > 12\% \quad \text{(Fora da margem)}
\end{cases}$$

Ao atingir 10 ou mais pontos, a equipe ativa é declarada vencedora e o estado da sala transiciona para `VITORIA`.

---

## 5. Endpoint HTTP Auxiliar: `/health`

- **Rota:** `GET /health`
- **Finalidade:** Monitoramento de liveness e health check de infraestrutura.
- **Resposta:**
  ```json
  {
    "status": "ok",
    "salasAtivas": 3,
    "uptime": "2h15m42s"
  }
  ```
$$
