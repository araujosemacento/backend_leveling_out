package entities

// Lances canônicos da disputa de Pedra, Papel e Tesoura
const (
	LancePedra   = "pedra"
	LancePapel   = "papel"
	LanceTesoura = "tesoura"
)

// Estados de resolução da disputa de iniciativa
const (
	StatusAguardandoOponente = "AGUARDANDO_OPONENTE"
	StatusAguardandoLances   = "AGUARDANDO_LANCES"
	StatusResolvido          = "RESOLVIDO"
	VencedorEmpate           = "EMPATE"
)

// PptMoveRequest modela o payload enviado pelo cliente ao lançar uma jogada
type PptMoveRequest struct {
	SalaCodigo string `json:"sala_codigo"`
	PlayerID   string `json:"player_id"`
	Equipe     string `json:"equipe"`
	Lance      string `json:"lance"`
	Rodada     int    `json:"rodada"`
}

// PptResultResponse modela o resultado divulgado após a resolução do confronto
type PptResultResponse struct {
	Status    string            `json:"status"`
	Rodada    int               `json:"rodada"`
	Vencedor  string            `json:"vencedor"`
	Resultado string            `json:"resultado"`
	Lances    map[string]string `json:"lances"`
}

// PptStatusResponse modela a consulta do estado corrente da disputa
type PptStatusResponse struct {
	Status            string   `json:"status"`
	Rodada            int      `json:"rodada"`
	JogadoresEnviados int      `json:"jogadores_enviados,omitempty"`
	EquipesEnviadas   []string `json:"equipes_enviadas"`
}
