# Diretrizes de Segurança e Blindagem do Backend

Este documento estabelece a arquitetura de defesa em profundidade, o diagnóstico do estado atual da base de código e as práticas recomendadas para mitigar riscos de injeções maliciosas, spoofing de cabeçalhos, ataques de negação de serviço (DoS/DDoS) e adulteração de estado no servidor PocketBase e no proxy reverso do jogo Leveling Out.

---

## 1. Princípio Central: Confiança Zero no Cliente (Zero Trust)

Aplicações web estáticas hospedadas no navegador (como o front-end em GitHub Pages) operam em ambiente público não confiável. Todas as variáveis de ambiente embutidas no pacote estático, URLs e cabeçalhos HTTP (incluindo chaves de proxy como X-Proxy-Secret) são visíveis para qualquer usuário via ferramentas de desenvolvedor.

Portanto, a segurança real da aplicação reside exclusivamente nas validações autoritativas executadas no proxy reverso e nos hooks internos do PocketBase.

---

## 2. O Paradoxo Criptográfico do Código Aberto

Um dilema recorrente em aplicações web públicas é a tentativa de fazer o front-end "criptografar" mensagens para que o servidor confie que o remetente é um cliente legítimo e impeça que terceiros com o código aberto repliquem suas requisições:

1. **A Inviabilidade Criptográfica no Cliente Aberto:**
   - Criptografia Assimétrica: se o front-end cifra com a chave pública do servidor, qualquer pessoa com acesso à chave pública pode cifrar dados arbitrários via terminal. Isso provê confidencialidade contra interceptadores, mas não autenticidade do remetente. Se o cliente usasse uma chave privada para assinar requisições, essa chave precisaria residir no JavaScript do navegador, tornando-se pública imediatamente.
   - Criptografia Simétrica: o segredo compartilhado ficaria exposto no pacote estático distribuído.
   - Esse limite é formalmente conhecido como o Problema da Criptografia em Caixa-Branca (White-box Cryptography).

2. **Diferenciação entre Segredo de Proxy e Credencial Privada:**
   - O cabeçalho X-Proxy-Secret funciona como token público de identificação da aplicação (semelhante às chaves públicas do Firebase ou Supabase).
   - Sua função é filtrar tráfego bruto automatizado (scanners de portas, robôs de varredura e requisições acidentais), e não restringir usuários técnicos com ferramentas de console.

3. **Soluções Complementares da Indústria:**
   - Atestação de Navegador via Cloudflare Turnstile: executa desafios não intrusivos em tempo de execução para atestar a presença de um motor de navegador legítimo (V8, WebKit, Gecko), bloqueando chamadas diretas via cURL ou scripts em lote.
   - Desafios de Handshake Efêmeros (Challenge: Response): emissão de nonces assinados pelo servidor com validade curta para prevenir ataques de repetição (replay attacks).

4. **O Pilar do Desenvolvimento de Jogos: Autoritarismo de Estado:**
   - Em arquiteturas multiplayer, não se tenta impedir o envio de requisições pela rede; retira-se do cliente qualquer poder de decisão.
   - O cliente emite apenas intenções simples ("meu palpite é 72", "meu lance é pedra"). O servidor atua como fonte única da verdade, calculando distâncias, validando turnos e somando pontos.

---

## 3. Diagnóstico Atual da Base de Código (Auditoria de Maturidade)

Atualmente, o projeto encontra-se em um estágio intermediário de transição entre a prototipagem rápida e o endurecimento de produção:

1. **Pontos Fortes Já Implementados:**
   - Cálculo autoritativo de proximidade e pontuação graduada (+4, +3, +2, 0) centralizado em pb_hooks/game_rules.pb.js.
   - Sorteio da meta oculta (5% a 95%) controlado pelo servidor no evento onRecordCreateRequest/onRecordUpdateRequest.
   - Resolução da iniciativa de Pedra, Papel e Tesoura em memória thread-safe no servidor em pb_hooks/iniciativa.pb.js.
   - Higienização automática e purga em cascata de salas (15 min) e jogadores inativos (2 min) em pb_hooks/cleanup.pb.js.

2. **Brechas e Vulnerabilidades Pendentes:**
   - Vazamento de Meta Oculta: a coleção rodadas possui listRule e viewRule vazias (""), expondo o campo meta_oculta via API REST e eventos SSE para qualquer jogador com o DevTools aberto antes do envio do palpite.
   - Permissões Excessivas no Banco: a coleção salas possui updateRule aberta (""), permitindo que clientes forjem placar_a, placar_b e vencedor diretamente via requisições PATCH. As coleções rodadas e jogadores possuem deleteRule aberta (""), permitindo que participantes anônimos apaguem registros de salas ou expulsem oponentes.
   - Confiança em Identidade Forjada: os endpoints e hooks confiam no player_id recebido no corpo da requisição JSON, sem vínculo com tokens de sessão assinados pelo servidor.
   - Falta de Sanitização Textual: os campos dica e nome não possuem validações de comprimento máximo nem rejeição de tags HTML nos hooks de persistência.

---

## 4. Camadas de Blindagem e Boas Práticas

### Camada 1: Validação de Entrada e Prevenção de Injeções

1. **Restrição de Tamanho de Payload (Max Body Size):**
   Como as ações de jogo trafegam apenas strings curtas e números, o proxy reverso deve limitar o corpo de requisições a no máximo 64 KB (exemplo no Nginx: client_max_body_size 64k;). Payloads maiores devem ser descartados imediatamente na borda da rede.

2. **Sanitização e Limite de Caracteres nos Hooks:**
   Nos hooks de persistência (onRecordCreateRequest e onRecordUpdateRequest):
   - Limitar dica a no máximo 60 caracteres.
   - Limitar nome de jogador a no máximo 20 caracteres.
   - Limitar codigo de sala a no máximo 16 caracteres alfanuméricos em caixa alta.
   - Rejeitar caracteres de controle e tags como <script> ou HTML cru para mitigar riscos de XSS no cliente de outros participantes.

3. **Validação de Limites Numéricos e Enumerações:**
   - Palpite: forçar número inteiro compreendido estritamente no intervalo de 0 a 100.
   - Lance de Pedra, Papel e Tesoura: validar pertinência exclusiva ao conjunto ['pedra', 'papel', 'tesoura'].
   - Queries parametrizadas: empregar sempre sintaxe de parâmetros preparados do PocketBase ({:param}) em consultas SQLite, nunca concatenando strings cruas.

### Camada 2: Blindagem contra Spoofing e Trapaças de Estado

1. **Cálculo Autoritativo no Servidor:**
   O front-end nunca calcula nem envia pontuações. O cliente transmite apenas o valor numérico do palpite. O cálculo da margem de erro, atribuição de pontos e transição de rodadas ocorrem exclusivamente no hook game_rules.pb.js.

2. **Prevenção de Ações Duplas ou Fora de Turno:**
   O servidor deve rejeitar requisições de lances ou palpites emitidos por jogadores que não correspondam à equipe ativa da rodada ou que já tenham submetido sua jogada para o ciclo vigente.

3. **Higienização de Cabeçalhos de Rede no Proxy:**
   - Sobrescrever X-Real-IP e X-Forwarded-For com o endereço IP real da conexão TCP de entrada, ignorando valores previamente injetados pelo cliente.
   - Validar estritamente o cabeçalho Origin contra a origem autorizada https://araujosemacento.github.io.

### Camada 3: Mitigação de DDoS e Esgotamento de Recursos

1. **Teto de Conexões Simultâneas por IP na Rota SSE:**
   A rota de streaming /api/realtime mantém conexões HTTP abertas continuamente. O proxy reverso deve impor limite de conexões simultâneas por endereço IP (exemplo: limit_conn no Nginx com teto de 6 conexões simultâneas por IP) para impedir esgotamento de descritores de arquivo.

2. **Limitação de Taxa (Rate Limiting) em Escrita:**
   Aplicar teto de requisições por segundo para métodos POST, PATCH e DELETE (por exemplo: 10 a 20 requisições por segundo por IP com margem controlada de burst). Descartar atualizações de presença (heartbeat) do mesmo jogador com intervalo inferior a 3 segundos.

3. **Teto Global de Salas Ativas (Proteção de Disco):**
   Adicionar validação na criação de novas salas. Caso o total de salas ativas atinja o limite máximo operacional (exemplo: 200 salas simultâneas), o servidor deve retornar status 429 Too Many Requests temporariamente.

4. **Higienização Automática Contínua (Zero Bloat):**
   Manter a rotina cron programada em cleanup.pb.js purgando jogadores inativos há mais de 2 minutos e salas sem movimentação há mais de 15 minutos em cascata.

### Camada 4: Sigilo de Dados em Trânsito (Meta Oculta)

1. **Omissão da Meta Oculta nas Fases Ativas:**
   O valor percentual do tubo de ensaio (meta_oculta) não deve ser exposto a nenhum participante exceto o Codificador da vez enquanto a rodada estiver nas fases DICA, ESCOLHA_ESPECTRO ou PALPITE.

2. **Entrega Condicional:**
   Retornar meta_oculta como nulo ou zero nas consultas dos demais clientes, liberando o valor real para ambos os times unicamente após a transição da rodada para a fase REVELADA.

### Camada 5: Permissões Mínimas no Banco (API Rules)

1. **Exclusão Proibida para Clientes Anônimos:**
   Definir deleteRule como null nas coleções salas, rodadas e jogadores no esquema do banco. Toda e qualquer exclusão deve ser disparada unicamente pelo script interno de limpeza.

2. **Proteção de Campos de Governança:**
   Campos críticos de controle de jogo (como placar_a, placar_b, vencedor e rodada_atual) devem ser alterados somente pela lógica interna do backend (hooks), bloqueando alterações manuais diretas via PATCH vindo do navegador.

---

## 5. Roteiro Prático de Implementação (Roadmap para Agentes e Desenvolvedores)

Para elevar o backend ao estado seguro preconizado por este guia, execute a seguinte sequência de tarefas:

1. **Ajuste Declarativo de Migrações (pb_migrations/):**
   - Alterar deleteRule para null em salas, rodadas e jogadores.
   - Definir updateRule de salas como null para clientes anônimos, delegando manipulações de placar a hooks ou rotas autoritativas.

2. **Ocultação Dinâmica da Meta nos Hooks (pb_hooks/game_rules.pb.js):**
   - Implementar interceptor de leitura para mascarar meta_oculta enquanto a rodada não atingir o estado REVELADA.

3. **Validação de Entrada nos Hooks (pb_hooks/game_rules.pb.js):**
   - Validar limites de caracteres e sanitização de strings para dica, nome e codigo.
   - Validar se o palpite submetido é um número inteiro entre 0 e 100.

4. **Configuração de Borda no Proxy Reverso (proxy-reverso/):**
   - Configurar limitação de conexões SSE concorrentes em /api/realtime.
   - Definir client_max_body_size em 64k.
   - Aplicar rate limiting por IP para requisições de escrita.
