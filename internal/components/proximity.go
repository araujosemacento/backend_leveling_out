package components

// Tiers de proximidade com suas respectivas pontuações canônicas
const (
	TierNaMosca     = "NA_MOSCA"
	TierMuitoPerto  = "MUITO_PERTO"
	TierPerto       = "PERTO"
	TierForaMargem  = "FORA_DA_MARGEM"
)

// ProximityResult encapsula o resultado avaliado pelo algoritmo de proximidade
type ProximityResult struct {
	Pontos int    `json:"pontos"`
	Diff   int    `json:"diff"`
	Tier   string `json:"tier"`
	Label  string `json:"label"`
}

// ProximityComponent modela o RegraProximidadeComponent no backend
type ProximityComponent struct{}

// NewProximityComponent instancia o componente de cálculo de proximidade
func NewProximityComponent() *ProximityComponent {
	return &ProximityComponent{}
}

// Calculate avalia a tolerância percentual absoluta e retorna a pontuação autoritativa
// Faixas:
// diff <= 2: 4 pontos (Na mosca)
// diff <= 6: 3 pontos (Muito perto)
// diff <= 12: 2 pontos (Perto)
// diff > 12: 0 pontos (Fora da margem)
func (c *ProximityComponent) Calculate(meta, guess int) ProximityResult {
	diff := meta - guess
	if diff < 0 {
		diff = -diff
	}

	switch {
	case diff <= 2:
		return ProximityResult{
			Pontos: 4,
			Diff:   diff,
			Tier:   TierNaMosca,
			Label:  "Na mosca!",
		}
	case diff <= 6:
		return ProximityResult{
			Pontos: 3,
			Diff:   diff,
			Tier:   TierMuitoPerto,
			Label:  "Muito perto!",
		}
	case diff <= 12:
		return ProximityResult{
			Pontos: 2,
			Diff:   diff,
			Tier:   TierPerto,
			Label:  "Perto!",
		}
	default:
		return ProximityResult{
			Pontos: 0,
			Diff:   diff,
			Tier:   TierForaMargem,
			Label:  "Fora da margem",
		}
	}
}
