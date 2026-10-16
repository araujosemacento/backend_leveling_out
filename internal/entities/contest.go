package entities

// Lances canônicos da disputa de Pedra, Papel e Tesoura
const (
	LancePedra   = "pedra"
	LancePapel   = "papel"
	LanceTesoura = "tesoura"
)

// Subfases da disputa de iniciativa (campo ppt_estado da sala)
const (
	PptEstadoEscolha   = "ESCOLHA"
	PptEstadoContagem  = "CONTAGEM"
	PptEstadoResultado = "RESULTADO"
)

// Resultados canônicos da arbitragem
const (
	PptResultadoEmpate   = "EMPATE"
	PptResultadoVitoriaA = "VITORIA_A"
	PptResultadoVitoriaB = "VITORIA_B"
)

// PptMoveRequest modela o payload enviado pelo representante ao lançar sua jogada.
// A equipe e o jogador são derivados das credenciais da requisição (Zero Trust).
type PptMoveRequest struct {
	Lance string `json:"lance"`
}

// PptResultResponse modela o resultado arbitrado do confronto
type PptResultResponse struct {
	Resultado string            `json:"resultado"`
	Vencedor  string            `json:"vencedor"`
	Lances    map[string]string `json:"lances"`
}

// LanceValido indica se o lance informado pertence ao conjunto canônico
func LanceValido(lance string) bool {
	return lance == LancePedra || lance == LancePapel || lance == LanceTesoura
}
