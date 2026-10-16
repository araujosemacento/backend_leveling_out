package protocol

import "time"

// Tipos de mensagens de ENTRADA (Cliente -> Servidor)
const (
	TipoEscolherEquipe   = "ESCOLHER_EQUIPE"
	TipoIniciarPartida   = "INICIAR_PARTIDA"
	TipoExpulsarJogador  = "EXPULSAR_JOGADOR"
	TipoBanirJogador     = "BANIR_JOGADOR"
	TipoDesfazerPartida  = "DESFAZER_PARTIDA"
	TipoLancePpt         = "LANCE_PPT"
	TipoEscolherEspectro = "ESCOLHER_ESPECTRO"
	TipoSubmeterDica     = "SUBMETER_DICA"
	TipoSliderMover      = "SLIDER_MOVER"
	TipoConfirmarPalpite = "CONFIRMAR_PALPITE"
	TipoRevanche         = "REVANCHE"
)

// Tipos de mensagens de SAÍDA (Servidor -> Cliente)
const (
	TipoBemVindo     = "BEM_VINDO"
	TipoSync         = "SYNC"
	TipoSliderStream = "SLIDER_STREAM"
	TipoNotificacao  = "NOTIFICACAO"
	TipoKickado      = "KICKADO"
	TipoBanido       = "BANIDO"
	TipoErro         = "ERRO"
)

// Códigos de status de fechamento de WebSocket dedicados à aplicação (RFC 6455 faixa 4000-4999)
const (
	StatusKickado = 4001 // Jogador expulso pelo anfitrião
)

// MensagemIn encapsula o envelope de entrada genérico recebido pelo WebSocket
type MensagemIn struct {
	Tipo    string                 `json:"tipo"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// PayloadEscolherEquipe decodifica a ação de alocação de equipe
type PayloadEscolherEquipe struct {
	Equipe string `json:"equipe"`
}

// PayloadExpulsarJogador decodifica a moderação do anfitrião
type PayloadExpulsarJogador struct {
	TargetPlayerID string `json:"targetPlayerId"`
}

// PayloadLancePpt decodifica o lance do representante
type PayloadLancePpt struct {
	Lance string `json:"lance"`
}

// PayloadEscolherEspectro decodifica a seleção do Codificador
type PayloadEscolherEspectro struct {
	CartaID  string `json:"cartaId,omitempty"`
	Esquerda string `json:"esquerda,omitempty"`
	Direita  string `json:"direita,omitempty"`
}

// PayloadSubmeterDica decodifica a pista textual do Codificador
type PayloadSubmeterDica struct {
	Dica string `json:"dica"`
}

// PayloadSliderMover decodifica o arrasto contínuo do Palpiteiro
type PayloadSliderMover struct {
	Posicao float64 `json:"posicao"`
}

// Envelope de saída padronizado
type MensagemOut struct {
	Tipo    string      `json:"tipo"`
	Payload interface{} `json:"payload"`
}

// MsgBemVindo entrega as credenciais geradas no handshake
type MsgBemVindo struct {
	PlayerID     string `json:"playerId"`
	SessionToken string `json:"sessionToken"`
	IsHost       bool   `json:"isHost"`
	Nome         string `json:"nome"`
	SalaCodigo   string `json:"salaCodigo"`
}

// SnapshotJogador representa a visão de um jogador no snapshot
type SnapshotJogador struct {
	ID          string    `json:"id"`
	PlayerID    string    `json:"playerId"`
	Nome        string    `json:"nome"`
	Equipe      string    `json:"equipe"`
	Papel       string    `json:"papel"`
	IsHost      bool      `json:"isHost"`
	IsOnline    bool      `json:"isOnline"`
	EquipeDesde time.Time `json:"equipeDesde"`
}

// SnapshotPpt representa o estado da disputa projetado para um participante
type SnapshotPpt struct {
	Estado          string     `json:"estado"` // ESCOLHA, CONTAGEM, RESULTADO
	Prazo           *time.Time `json:"prazo,omitempty"`
	RevelacaoEm     *time.Time `json:"revelacaoEm,omitempty"`
	EnviouA         bool       `json:"enviouA"`
	EnviouB         bool       `json:"enviouB"`
	RepresentanteA  string     `json:"representanteA"`
	RepresentanteB  string     `json:"representanteB"`
	Resultado       string     `json:"resultado,omitempty"`
	LanceA          string     `json:"lanceA,omitempty"`
	LanceB          string     `json:"lanceB,omitempty"`
	MeuLance        string     `json:"meuLance,omitempty"` // Disponível apenas para o representante da própria equipe
}

// SnapshotRodada representa o estado da rodada ativa projetado
type SnapshotRodada struct {
	Numero           int     `json:"numero"`
	Equipe           string  `json:"equipe"`
	CodificadorID    string  `json:"codificadorId"`
	PalpiteiroID     string  `json:"palpiteiroId"`
	EspectroEsquerda string  `json:"espectroEsquerda,omitempty"`
	EspectroDireita  string  `json:"espectroDireita,omitempty"`
	MetaOculta       *int    `json:"metaOculta,omitempty"` // Nulo para quem não for Codificador até REVELACAO
	Dica             string  `json:"dica,omitempty"`
	Palpite          *int    `json:"palpite,omitempty"`
	Pontos           int     `json:"pontos"`
	Tier             string  `json:"tier,omitempty"`
	Label            string  `json:"label,omitempty"`
}

// SnapshotSalaProjetado é o modelo canônico de FULL_SYNC entregue ao GameEngine
type SnapshotSalaProjetado struct {
	Codigo                  string            `json:"codigo"`
	Fase                    string            `json:"fase"`
	Rodada                  int               `json:"rodada"`
	EquipeAtiva             string            `json:"equipeAtiva"`
	PlacarA                 int               `json:"placarA"`
	PlacarB                 int               `json:"placarB"`
	Vencedor                string            `json:"vencedor,omitempty"`
	HostID                  string            `json:"hostId"`
	MotivoFim               string            `json:"motivoFim,omitempty"`
	Jogadores               []SnapshotJogador `json:"jogadores"`
	Ppt                     SnapshotPpt       `json:"ppt"`
	RodadaAtiva             SnapshotRodada    `json:"rodadaAtiva"`
	CartasEspectroOpcoes    []interface{}     `json:"cartasOpcoes,omitempty"` // Duas opções para ESCOLHA_ESPECTRO
	DesconectadosRecentes   []string          `json:"desconectados,omitempty"`
}
