package entities

import "time"

// Identificadores canônicos das equipes sob os estandartes de animais
const (
	EquipePato       = "A"
	EquipeGanso      = "B"
	EquipeAleatorio  = "ALEATORIO"
	EquipeEspectador = "ESPECTADOR"
)

// Papéis dinâmicos exercidos por um participante na rodada
const (
	PapelCodificador = "CODIFICADOR"
	PapelPalpiteiro  = "PALPITEIRO"
	PapelEspectador  = "ESPECTADOR"
)

// Jogador modela o participante individual conectado à sala (EntidadeParticipante)
type Jogador struct {
	ID           string    `json:"id"`
	SalaCodigo   string    `json:"salaCodigo"`
	PlayerID     string    `json:"playerId"`
	Nome         string    `json:"nome"`
	Equipe       string    `json:"equipe"`
	Papel        string    `json:"papel"`
	IsHost       bool      `json:"isHost"`
	IsOnline     bool      `json:"isOnline"`
	SessionToken string    `json:"-"` // Oculto em serializações públicas (Zero Trust)
	EquipeDesde  time.Time `json:"equipeDesde"`
	LastSeen     time.Time `json:"lastSeen"`
}

// CloneProjetado gera uma cópia defensiva do jogador sem campos sigilosos
func (j *Jogador) CloneProjetado() Jogador {
	return Jogador{
		ID:          j.ID,
		SalaCodigo:  j.SalaCodigo,
		PlayerID:    j.PlayerID,
		Nome:        j.Nome,
		Equipe:      j.Equipe,
		Papel:       j.Papel,
		IsHost:      j.IsHost,
		IsOnline:    j.IsOnline,
		EquipeDesde: j.EquipeDesde,
		LastSeen:    j.LastSeen,
	}
}
