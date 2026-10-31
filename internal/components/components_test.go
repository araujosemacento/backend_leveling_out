package components_test

import (
	"fmt"
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

func TestSpectrumsComponentKnuthShuffleAndDiscard(t *testing.T) {
	spec := components.NewSpectrumsComponent()

	if spec.TotalCatalogo() != 150 {
		t.Fatalf("TotalCatalogo() = %d; want 150", spec.TotalCatalogo())
	}
	if spec.RestantesNoMonte() != 150 {
		t.Fatalf("RestantesNoMonte() inicial = %d; want 150", spec.RestantesNoMonte())
	}
	if spec.TotalNoDescarte() != 0 {
		t.Fatalf("TotalNoDescarte() inicial = %d; want 0", spec.TotalNoDescarte())
	}

	vistos := make(map[string]bool, 150)
	porCategoria := make(map[string]int, 5)

	// 75 sorteios de pares consomem exatamente todas as 150 cartas sem nenhuma repetição
	for i := 0; i < 75; i++ {
		c1, c2 := spec.SortearPar()
		if c1.ID == c2.ID {
			t.Fatalf("rodada %d: SortearPar retornou cartas idênticas (%s)", i+1, c1.ID)
		}
		if vistos[c1.ID] {
			t.Fatalf("rodada %d: carta %s repetida antes de esgotar o baralho", i+1, c1.ID)
		}
		if vistos[c2.ID] {
			t.Fatalf("rodada %d: carta %s repetida antes de esgotar o baralho", i+1, c2.ID)
		}
		vistos[c1.ID] = true
		vistos[c2.ID] = true
		porCategoria[c1.Categoria]++
		porCategoria[c2.Categoria]++
	}

	if len(vistos) != 150 {
		t.Fatalf("cartas únicas sorteadas = %d; want 150", len(vistos))
	}
	if spec.RestantesNoMonte() != 0 {
		t.Fatalf("RestantesNoMonte() após 75 pares = %d; want 0", spec.RestantesNoMonte())
	}
	if spec.TotalNoDescarte() != 150 {
		t.Fatalf("TotalNoDescarte() após 75 pares = %d; want 150", spec.TotalNoDescarte())
	}

	categoriasEsperadas := []string{
		components.CategoriaSensorial,
		components.CategoriaJuizoValor,
		components.CategoriaComportamento,
		components.CategoriaComplexidade,
		components.CategoriaCulturaPop,
	}
	for _, cat := range categoriasEsperadas {
		if porCategoria[cat] != 30 {
			t.Errorf("categoria %q possui %d cartas; want 30", cat, porCategoria[cat])
		}
	}

	// 76º sorteio deve reciclar a pilha de descarte e reembaralhar via Knuth Shuffle
	c1, c2 := spec.SortearPar()
	if c1.ID == c2.ID {
		t.Fatalf("após reciclagem: SortearPar retornou cartas idênticas (%s)", c1.ID)
	}
	if spec.RestantesNoMonte() != 148 {
		t.Fatalf("RestantesNoMonte() após reciclagem e 1 par = %d; want 148", spec.RestantesNoMonte())
	}
	if spec.TotalNoDescarte() != 2 {
		t.Fatalf("TotalNoDescarte() após reciclagem e 1 par = %d; want 2", spec.TotalNoDescarte())
	}

	// Validação de cartas canônicas adicionadas / refinadas
	cartasEsperadas := map[string]struct{ esq, dir string }{
		"c29":  {"Insosso", "Salgado"},
		"c51":  {"Marcante", "Insignificante"},
		"c57":  {"Sucesso", "Fracasso"},
		"c58":  {"Santo", "Profano"},
		"c89":  {"Comida rotineira", "Manjar dos deuses"},
		"c103": {"Acessível", "Inacessível"},
		"c106": {"Amador", "Profissional"},
		"c116": {"Concreto", "Abstrato"},
		"c133": {"Sidequest", "Boss Battle"},
		"c138": {"O Lado Puro da Força", "O Lado Negro da Força"},
	}
	for id, esp := range cartasEsperadas {
		carta, achou := spec.ObterPorID(id)
		if !achou {
			t.Errorf("carta %s não encontrada no catálogo", id)
			continue
		}
		if carta.Esquerda != esp.esq || carta.Direita != esp.dir {
			t.Errorf("carta %s = %s / %s; want %s / %s", id, carta.Esquerda, carta.Direita, esp.esq, esp.dir)
		}
	}
}

func TestSpectrumsComponentIntegridadeECatalogoDefinitivo(t *testing.T) {
	spec := components.NewSpectrumsComponent()

	// 1. Testa busca por ID inexistente
	if _, achou := spec.ObterPorID("c999"); achou {
		t.Error("ObterPorID('c999') deveria retornar false para ID inexistente")
	}

	// 2. Validação exaustiva de todas as 150 cartas do acervo canônico
	idsVistos := make(map[string]bool, 150)
	polosVistos := make(map[string]string, 300)

	for i := 1; i <= 150; i++ {
		id := fmt.Sprintf("c%d", i)

		carta, achou := spec.ObterPorID(id)
		if !achou {
			t.Fatalf("carta canônica %s não encontrada no catálogo definitivo", id)
		}

		if idsVistos[carta.ID] {
			t.Errorf("ID duplicado no catálogo definitivo: %s", carta.ID)
		}
		idsVistos[carta.ID] = true

		if carta.Esquerda == "" || carta.Direita == "" {
			t.Errorf("carta %s possui polos em branco: '%s' / '%s'", carta.ID, carta.Esquerda, carta.Direita)
		}
		if carta.Esquerda == carta.Direita {
			t.Errorf("carta %s possui polos idênticos: '%s'", carta.ID, carta.Esquerda)
		}

		if anterior, existe := polosVistos[carta.Esquerda]; existe {
			t.Errorf("polo esquerdo '%s' da carta %s já usado na carta %s", carta.Esquerda, carta.ID, anterior)
		}
		polosVistos[carta.Esquerda] = carta.ID

		if anterior, existe := polosVistos[carta.Direita]; existe {
			t.Errorf("polo direito '%s' da carta %s já usado na carta %s", carta.Direita, carta.ID, anterior)
		}
		polosVistos[carta.Direita] = carta.ID
	}

	if len(idsVistos) != 150 {
		t.Fatalf("esperado 150 IDs únicos no catálogo definitivo, obtido %d", len(idsVistos))
	}
	if len(polosVistos) != 300 {
		t.Fatalf("esperado 300 polos textuais únicos no catálogo definitivo, obtido %d", len(polosVistos))
	}
}
