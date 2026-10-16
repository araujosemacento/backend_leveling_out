package components


// SpectrumCard modela uma carta de espectro conceitual com dois polos opostos
type SpectrumCard struct {
	ID       string `json:"id"`
	Esquerda string `json:"esquerda"`
	Direita  string `json:"direita"`
}

// SpectrumsComponent modela o PolosSemanticosComponent: catálogo atômico e sorteador de cartas
type SpectrumsComponent struct {
	cartas []SpectrumCard
}

// NewSpectrumsComponent inicializa o catálogo com o acervo canônico do jogo
func NewSpectrumsComponent() *SpectrumsComponent {
	return &SpectrumsComponent{
		cartas: []SpectrumCard{
			{ID: "c1", Esquerda: "Anônimo", Direita: "Famoso"},
			{ID: "c2", Esquerda: "Gelado", Direita: "Fervendo"},
			{ID: "c3", Esquerda: "Inofensivo", Direita: "Perigoso"},
			{ID: "c4", Esquerda: "Triste", Direita: "Feliz"},
			{ID: "c5", Esquerda: "Frágil", Direita: "Indestrutível"},
			{ID: "c6", Esquerda: "Barato", Direita: "Luxuoso"},
			{ID: "c7", Esquerda: "Inútil", Direita: "Indispensável"},
			{ID: "c8", Esquerda: "Subestimado", Direita: "Superestimado"},
			{ID: "c9", Esquerda: "Fácil de aprender", Direita: "Difícil de dominar"},
			{ID: "c10", Esquerda: "Normal", Direita: "Bizarro"},
			{ID: "c11", Esquerda: "Calmo", Direita: "Estressante"},
			{ID: "c12", Esquerda: "Silencioso", Direita: "Barulhento"},
			{ID: "c13", Esquerda: "Saboroso", Direita: "Intragável"},
			{ID: "c14", Esquerda: "Antigo", Direita: "Futurista"},
			{ID: "c15", Esquerda: "Legal", Direita: "Ilegal"},
		},
	}
}

// SortearPar escolhe duas cartas distintas para o Codificador escolher
func (s *SpectrumsComponent) SortearPar() (SpectrumCard, SpectrumCard) {
	total := len(s.cartas)
	if total < 2 {
		return SpectrumCard{ID: "c1", Esquerda: "Esquerda", Direita: "Direita"},
			SpectrumCard{ID: "c2", Esquerda: "Polo A", Direita: "Polo B"}
	}

	idx1 := sorteioSeguro(total)
	idx2 := sorteioSeguro(total - 1)
	if idx2 >= idx1 {
		idx2++
	}

	return s.cartas[idx1], s.cartas[idx2]
}

// ObterPorID busca uma carta específica pelo identificador
func (s *SpectrumsComponent) ObterPorID(id string) (SpectrumCard, bool) {
	for _, c := range s.cartas {
		if c.ID == id {
			return c, true
		}
	}
	return SpectrumCard{}, false
}

