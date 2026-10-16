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
	sala := "TEST_SALA"
	arbiter.Abrir(sala, "repA", "repB")

	// Apenas o representante da equipe pode jogar
	if _, err := arbiter.SubmitMove(sala, "intruso", entities.EquipePato, entities.LancePedra); err != components.ErrNaoRepresentante {
		t.Fatalf("esperado ErrNaoRepresentante, recebido %v", err)
	}
	if _, err := arbiter.SubmitMove(sala, "repA", entities.EquipePato, "lagarto"); err != components.ErrLanceInvalido {
		t.Fatalf("esperado ErrLanceInvalido, recebido %v", err)
	}

	// 1. Lance da Equipe Pato
	pronto, err := arbiter.SubmitMove(sala, "repA", entities.EquipePato, entities.LancePedra)
	if err != nil || pronto {
		t.Fatalf("primeiro lance: pronto=%v err=%v", pronto, err)
	}

	// Lance definitivo: não permite troca
	if _, err := arbiter.SubmitMove(sala, "repA", entities.EquipePato, entities.LanceTesoura); err != components.ErrLanceDefinitivo {
		t.Fatalf("esperado ErrLanceDefinitivo, recebido %v", err)
	}
	if got := arbiter.LanceDaEquipe(sala, entities.EquipePato); got != entities.LancePedra {
		t.Errorf("lance da equipe alterado: %s", got)
	}

	// 2. Lance da Equipe Ganso: tesoura contra pedra, vitória de Pato
	pronto, err = arbiter.SubmitMove(sala, "repB", entities.EquipeGanso, entities.LanceTesoura)
	if err != nil || !pronto {
		t.Fatalf("segundo lance: pronto=%v err=%v", pronto, err)
	}
	res, err := arbiter.Arbitrar(sala)
	if err != nil {
		t.Fatalf("Arbitrar erro: %v", err)
	}
	if res.Vencedor != entities.EquipePato || res.Resultado != entities.PptResultadoVitoriaA {
		t.Errorf("vencedor esperado Pato, recebido %s/%s", res.Vencedor, res.Resultado)
	}
}

func TestPptArbiterPreencherTimeout(t *testing.T) {
	arbiter := components.NewPptArbiterComponent()
	sala := "TIMEOUT"
	arbiter.Abrir(sala, "repA", "repB")

	_, _ = arbiter.SubmitMove(sala, "repA", entities.EquipePato, entities.LancePapel)
	if !arbiter.Preencher(sala, entities.EquipePato, entities.EquipeGanso) {
		t.Fatal("preenchimento deveria completar a disputa")
	}
	if got := arbiter.LanceDaEquipe(sala, entities.EquipePato); got != entities.LancePapel {
		t.Errorf("preenchimento não deve sobrescrever lance existente, recebido %s", got)
	}
	if !entities.LanceValido(arbiter.LanceDaEquipe(sala, entities.EquipeGanso)) {
		t.Error("lance sorteado deve ser válido")
	}
}

func TestDecidirPpt(t *testing.T) {
	cases := []struct{ a, b, resultado, vencedor string }{
		{entities.LancePedra, entities.LancePedra, entities.PptResultadoEmpate, ""},
		{entities.LancePedra, entities.LanceTesoura, entities.PptResultadoVitoriaA, entities.EquipePato},
		{entities.LancePapel, entities.LancePedra, entities.PptResultadoVitoriaA, entities.EquipePato},
		{entities.LanceTesoura, entities.LancePapel, entities.PptResultadoVitoriaA, entities.EquipePato},
		{entities.LanceTesoura, entities.LancePedra, entities.PptResultadoVitoriaB, entities.EquipeGanso},
	}
	for _, c := range cases {
		r, v := components.DecidirPpt(c.a, c.b)
		if r != c.resultado || v != c.vencedor {
			t.Errorf("DecidirPpt(%s,%s) = %s/%s; want %s/%s", c.a, c.b, r, v, c.resultado, c.vencedor)
		}
	}
}
