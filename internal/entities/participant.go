package entities

import "time"

// Identificadores canônicos das equipes sob os estandartes de animais
const (
	EquipeCorvo      = "A"
	EquipeAguia      = "B"
	EquipeEspectador = "ESPECTADOR"
)

// Papéis dinâmicos exercidos por um participante na rodada
const (
	PapelCodificador = "CODIFICADOR"
	PapelPalpiteiro  = "PALPITEIRO"
	PapelEspectador  = "ESPECTADOR"
)

// Jogador modela o participante individual conectado à sala
type Jogador struct {
	ID         string    `json:"id" db:"id"`
	SalaCodigo string    `json:"sala_codigo" db:"sala_codigo"`
	PlayerID   string    `json:"player_id" db:"player_id"`
	Nome       string    `json:"nome" db:"nome"`
	Equipe     string    `json:"equipe" db:"equipe"`
	Papel      string    `json:"papel" db:"papel"`
	LastSeen   time.Time `json:"last_seen" db:"last_seen"`
}
