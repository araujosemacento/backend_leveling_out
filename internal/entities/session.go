package entities

import "time"

// Fases canônicas da máquina de estados global (EntidadeSessao)
const (
	FaseLobby          = "LOBBY"
	FaseIniciativa     = "INICIATIVA"
	FaseRodadaEspectro = "ESCOLHA_ESPECTRO"
	FaseRodadaDica     = "RODADA_DICA"
	FaseRodadaPalpite  = "RODADA_PALPITE"
	FaseRodadaRevelada = "REVELACAO"
	FaseFimJogo        = "FIM_JOGO"
)

// MetaPontuacaoPadrao define a pontuação necessária para a vitória
const MetaPontuacaoPadrao = 10

// RodadaResumo armazena o histórico compacto de uma rodada concluída (para analytics finais)
type RodadaResumo struct {
	Numero           int     `json:"numero"`
	Equipe           string  `json:"equipe"`
	CodificadorID    string  `json:"codificadorId"`
	PalpiteiroID     string  `json:"palpiteiroId"`
	EspectroEsquerda string  `json:"espectroEsquerda"`
	EspectroDireita  string  `json:"espectroDireita"`
	MetaOculta       int     `json:"metaOculta"`
	Palpite          int     `json:"palpite"`
	Dica             string  `json:"dica"`
	Pontos           int     `json:"pontos"`
	Tier             string  `json:"tier"`
	Diff             int     `json:"diff"`
	DuracaoSegundos  float64 `json:"duracaoSegundos"`
}

// RodadaAtiva armazena os dados da rodada que está sendo jogada no momento
type RodadaAtiva struct {
	Numero              int       `json:"numero"`
	Equipe              string    `json:"equipe"`
	CodificadorID       string    `json:"codificadorId"`
	PalpiteiroID        string    `json:"palpiteiroId"`
	LiderRodadaID       string    `json:"liderRodadaId,omitempty"`       // Líder incumbido de escolher o espectro
	CartaSelecionadaID  string    `json:"cartaSelecionadaId,omitempty"`  // Carta pré-marcada pelo líder
	VotoParceiroCartaID string    `json:"votoParceiroCartaId,omitempty"` // Carta votada pelo parceiro (mãozinha)
	PrazoEspectro       time.Time `json:"prazoEspectro,omitempty"`       // Timeout de 30 segundos
	EspectroEsquerda    string    `json:"espectroEsquerda"`
	EspectroDireita     string    `json:"espectroDireita"`
	MetaOculta          int       `json:"metaOculta"` // Sigilosa até REVELACAO
	Dica                string    `json:"dica"`
	Palpite             int       `json:"palpite"`
	SliderPosicao       float64   `json:"sliderPosicao"` // Streaming dinâmico do arrasto
	Pontos              int       `json:"pontos"`
	Tier                string    `json:"tier"`
	Label               string    `json:"label"`
	IniciadaEm          time.Time `json:"iniciadaEm"`
	ReveladaEm          time.Time `json:"reveladaEm"`
}

// SalaEstado representa a instância operacional viva em memória da macro-sessão de jogo
type SalaEstado struct {
	Codigo      string                  `json:"codigo"`
	Fase        string                  `json:"fase"`
	RodadaAtual int                     `json:"rodada"`
	EquipeAtiva string                  `json:"equipeAtiva"`
	PlacarA     int                     `json:"placarA"`
	PlacarB     int                     `json:"placarB"`
	Vencedor    string                  `json:"vencedor,omitempty"`
	HostID      string                  `json:"hostId"`
	MotivoFim   string                  `json:"motivoFim,omitempty"`
	Ppt         PptContestState         `json:"ppt"`
	Rodada      RodadaAtiva             `json:"rodadaAtiva"`
	Historico   []RodadaResumo          `json:"historico,omitempty"`
	Jogadores   map[string]*Jogador     `json:"-"` // Mapeamento interno player_id -> Jogador
}

// PptContestState encapsula o estado público e temporal da disputa de iniciativa
type PptContestState struct {
	Estado         string    `json:"estado"` // ESCOLHA, CONTAGEM, RESULTADO
	Prazo          time.Time `json:"prazo"`
	RevelacaoEm    time.Time `json:"revelacaoEm"`
	EnviouA        bool      `json:"enviouA"`
	EnviouB        bool      `json:"enviouB"`
	RepresentanteA string    `json:"representanteA"`
	RepresentanteB string    `json:"representanteB"`
	Resultado      string    `json:"resultado,omitempty"`
	LanceA         string    `json:"lanceA,omitempty"` // Apenas após o Já!
	LanceB         string    `json:"lanceB,omitempty"` // Apenas após o Já!
}
