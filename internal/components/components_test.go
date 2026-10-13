package components_test

import (
	"testing"

	"leveling-out-backend/internal/components"
	"leveling-out-backend/internal/entities"
)

func TestProximityComponent(t *testing.T) {
	comp := components.NewProximityComponent()

	tests := []struct {
		name       string
		meta       int
		guess      int
		wantPontos int
		wantTier   string
	}{
		{"Exato (0% erro)", 50, 50, 4, components.TierNaMosca},
		{"Na mosca no limite (2% erro)", 50, 52, 4, components.TierNaMosca},
		{"Muito perto no início (3% erro)", 50, 53, 3, components.TierMuitoPerto},
		{"Muito perto no limite (6% erro)", 50, 56, 3, components.TierMuitoPerto},
		{"Perto no início (7% erro)", 50, 57, 2, components.TierPerto},
		{"Perto no limite (12% erro)", 50, 62, 2, components.TierPerto},
		{"Fora da margem (13% erro)", 50, 63, 0, components.TierForaMargem},
		{"Extremo superior fora", 10, 90, 0, components.TierForaMargem},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := comp.Calculate(tt.meta, tt.guess)
			if res.Pontos != tt.wantPontos {
				t.Errorf("Calculate(%d, %d).Pontos = %d; want %d", tt.meta, tt.guess, res.Pontos, tt.wantPontos)
			}
			if res.Tier != tt.wantTier {
				t.Errorf("Calculate(%d, %d).Tier = %s; want %s", tt.meta, tt.guess, res.Tier, tt.wantTier)
			}
		})
	}
}

func TestSecretTargetComponent(t *testing.T) {
	comp := components.NewSecretTargetComponent()

	for i := 0; i < 100; i++ {
		target := comp.Generate()
		if target < 5 || target > 95 {
			t.Fatalf("Generate() = %d; fora do intervalo seguro [5, 95]", target)
		}
	}
}

func TestPptArbiterComponent(t *testing.T) {
	arbiter := components.NewPptArbiterComponent()
	salaCodigo := "TEST_SALA"
	rodada := 1

	// 1. Lance da Equipe Corvo (A)
	res1, resolved1, err1 := arbiter.SubmitMove(entities.PptMoveRequest{
		SalaCodigo: salaCodigo,
		PlayerID:   "player1",
		Equipe:     entities.EquipeCorvo,
		Lance:      entities.LancePedra,
		Rodada:     rodada,
	})
	if err1 != nil {
		t.Fatalf("SubmitMove erro: %v", err1)
	}
	if resolved1 {
		t.Errorf("Rodada não deveria ser resolvida com apenas 1 lance")
	}
	if res1.Status != entities.StatusAguardandoOponente {
		t.Errorf("Status esperado %s, recebido %s", entities.StatusAguardandoOponente, res1.Status)
	}

	// 2. Lance da Equipe Águia (B) - Tesoura contra Pedra -> Vitória de Corvo (A)
	res2, resolved2, err2 := arbiter.SubmitMove(entities.PptMoveRequest{
		SalaCodigo: salaCodigo,
		PlayerID:   "player2",
		Equipe:     entities.EquipeAguia,
		Lance:      entities.LanceTesoura,
		Rodada:     rodada,
	})
	if err2 != nil {
		t.Fatalf("SubmitMove erro: %v", err2)
	}
	if !resolved2 {
		t.Errorf("Rodada deveria ter sido resolvida com 2 lances")
	}
	if res2.Vencedor != entities.EquipeCorvo {
		t.Errorf("Vencedor esperado %s, recebido %s", entities.EquipeCorvo, res2.Vencedor)
	}
}
