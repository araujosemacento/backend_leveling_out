package entities

// Fases canônicas do macro-ciclo da sala (EntidadeSessao)
const (
	FaseLobby      = "LOBBY"
	FaseIniciativa = "INICIATIVA"
	FaseEmRodada   = "EM_RODADA"
	FaseFimJogo    = "FIM_JOGO"
)

// Fases canônicas do micro-ciclo da rodada ativa (EntidadeSessao)
const (
	FaseRodadaEspectro  = "ESCOLHA_ESPECTRO"
	FaseRodadaDica      = "RODADA_DICA"
	FaseRodadaPalpite   = "RODADA_PALPITE"
	FaseRodadaRevelada  = "REVELACAO"
	FaseRodadaConcluida = "CONCLUIDA"
)

// MetaPontuacaoPadrao define a pontuação necessária para a vitória
const MetaPontuacaoPadrao = 10

// Sala representa a instância persistente da macro-sessão de jogo
type Sala struct {
	ID          string `json:"id" db:"id"`
	Codigo      string `json:"codigo" db:"codigo"`
	Fase        string `json:"fase" db:"fase"`
	RodadaAtual int    `json:"rodada_atual" db:"rodada_atual"`
	EquipeAtiva string `json:"equipe_ativa" db:"equipe_ativa"`
	PlacarA     int    `json:"placar_a" db:"placar_a"`
	PlacarB     int    `json:"placar_b" db:"placar_b"`
	Vencedor    string `json:"vencedor,omitempty" db:"vencedor"`
}

// Rodada representa a instância volátil do ciclo micro-temporal
type Rodada struct {
	ID               string `json:"id" db:"id"`
	SalaCodigo       string `json:"sala_codigo" db:"sala_codigo"`
	Numero           int    `json:"numero" db:"numero"`
	Equipe           string `json:"equipe" db:"equipe"`
	FaseRodada       string `json:"fase_rodada" db:"fase_rodada"`
	EspectroEsquerda string `json:"espectro_esquerda,omitempty" db:"espectro_esquerda"`
	EspectroDireita  string `json:"espectro_direita,omitempty" db:"espectro_direita"`
	MetaOculta       int    `json:"meta_oculta,omitempty" db:"meta_oculta"`
	Dica             string `json:"dica,omitempty" db:"dica"`
	Palpite          int    `json:"palpite,omitempty" db:"palpite"`
	Pontos           int    `json:"pontos" db:"pontos"`
}
