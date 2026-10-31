package components

import "sync"

// Categorias canônicas do catálogo de espectros
const (
	CategoriaSensorial     = "Sensoriais e Físicos"
	CategoriaJuizoValor    = "Juízos de Valor e Opinião"
	CategoriaComportamento = "Comportamento e Cotidiano"
	CategoriaComplexidade  = "Complexidade e Esforço"
	CategoriaCulturaPop    = "Cultura Pop e Ficcional"
)

// SpectrumCard modela uma carta de espectro conceitual com dois polos opostos
type SpectrumCard struct {
	ID        string `json:"id"`
	Categoria string `json:"categoria"`
	Esquerda  string `json:"esquerda"`
	Direita   string `json:"direita"`
}

// SpectrumsComponent modela o PolosSemanticosComponent: catálogo atômico,
// baralho embaralhado via Knuth Shuffle e pilha de descarte sem reposição prematura.
type SpectrumsComponent struct {
	mu       sync.Mutex
	catalogo []SpectrumCard
	monte    []SpectrumCard
	descarte []SpectrumCard
}

// NewSpectrumsComponent inicializa o catálogo com 150 espectros (30 por categoria)
// e embaralha o monte inicial utilizando o algoritmo de Knuth (Fisher-Yates).
func NewSpectrumsComponent() *SpectrumsComponent {
	catalogo := construirCatalogoCanonico()
	monte := make([]SpectrumCard, len(catalogo))
	copy(monte, catalogo)
	embaralharKnuth(monte)

	return &SpectrumsComponent{
		catalogo: catalogo,
		monte:    monte,
		descarte: make([]SpectrumCard, 0, len(catalogo)),
	}
}

// embaralharKnuth aplica o algoritmo clássico de Knuth (Fisher-Yates) in-place
// utilizando o gerador criptograficamente seguro do pacote.
func embaralharKnuth(cartas []SpectrumCard) {
	for i := len(cartas) - 1; i > 0; i-- {
		j := sorteioSeguro(i + 1)
		cartas[i], cartas[j] = cartas[j], cartas[i]
	}
}

// sacarDoMonte retira uma carta do topo do monte. Caso o monte esteja esgotado,
// a pilha de descarte é reciclada e reembaralhada via Knuth Shuffle em uma fatia isolada.
// Nota: a carta sacada NÃO é inserida no descarte aqui; quem consome o par
// é responsável por registrar ambas no descarte simultaneamente (evitando contaminação na reciclagem).
func (s *SpectrumsComponent) sacarDoMonte() SpectrumCard {
	if len(s.monte) == 0 {
		if len(s.descarte) > 0 {
			s.monte = make([]SpectrumCard, len(s.descarte))
			copy(s.monte, s.descarte)
			s.descarte = s.descarte[:0]
		} else {
			s.monte = make([]SpectrumCard, len(s.catalogo))
			copy(s.monte, s.catalogo)
		}
		embaralharKnuth(s.monte)
	}

	ultimoIdx := len(s.monte) - 1
	carta := s.monte[ultimoIdx]
	s.monte = s.monte[:ultimoIdx]
	return carta
}

// SortearPar retira duas cartas estritamente distintas do monte para o Codificador escolher.
// Ambas as cartas são adicionadas à pilha de descarte simultaneamente após a retirada,
// garantindo que mesmo se uma reciclagem for acionada entre o primeiro e o segundo saque,
// a primeira carta ainda não esteja no descarte e seja matematicamente impossível sair duplicada.
func (s *SpectrumsComponent) SortearPar() (SpectrumCard, SpectrumCard) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.catalogo) < 2 {
		return SpectrumCard{ID: "c1", Esquerda: "Esquerda", Direita: "Direita"},
			SpectrumCard{ID: "c2", Esquerda: "Polo A", Direita: "Polo B"}
	}

	c1 := s.sacarDoMonte()
	c2 := s.sacarDoMonte()

	s.descarte = append(s.descarte, c1, c2)
	return c1, c2
}

// ObterPorID busca uma carta específica pelo identificador no catálogo completo
func (s *SpectrumsComponent) ObterPorID(id string) (SpectrumCard, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, c := range s.catalogo {
		if c.ID == id {
			return c, true
		}
	}
	return SpectrumCard{}, false
}

// TotalCatalogo retorna a quantidade total de cartas cadastradas no acervo
func (s *SpectrumsComponent) TotalCatalogo() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.catalogo)
}

// RestantesNoMonte retorna quantas cartas ainda restam no monte antes da próxima reciclagem
func (s *SpectrumsComponent) RestantesNoMonte() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.monte)
}

// TotalNoDescarte retorna quantas cartas já foram consumidas e estão na pilha de descarte
func (s *SpectrumsComponent) TotalNoDescarte() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.descarte)
}

func construirCatalogoCanonico() []SpectrumCard {
	return []SpectrumCard{
		// =========================================================================
		// 1. Sensoriais e Físicos (30 espectros: c1 a c30)
		// =========================================================================
		{ID: "c1", Categoria: CategoriaSensorial, Esquerda: "Gelado", Direita: "Fervendo"},
		{ID: "c2", Categoria: CategoriaSensorial, Esquerda: "Silencioso", Direita: "Barulhento"},
		{ID: "c3", Categoria: CategoriaSensorial, Esquerda: "Frágil", Direita: "Indestrutível"},
		{ID: "c4", Categoria: CategoriaSensorial, Esquerda: "Leve", Direita: "Pesado"},
		{ID: "c5", Categoria: CategoriaSensorial, Esquerda: "Lento", Direita: "Veloz"},
		{ID: "c6", Categoria: CategoriaSensorial, Esquerda: "Macio", Direita: "Áspero"},
		{ID: "c7", Categoria: CategoriaSensorial, Esquerda: "Escuro", Direita: "Brilhante"},
		{ID: "c8", Categoria: CategoriaSensorial, Esquerda: "Seco", Direita: "Molhado"},
		{ID: "c9", Categoria: CategoriaSensorial, Esquerda: "Doce", Direita: "Amargo"},
		{ID: "c10", Categoria: CategoriaSensorial, Esquerda: "Suave", Direita: "Intenso"},
		{ID: "c11", Categoria: CategoriaSensorial, Esquerda: "Minúsculo", Direita: "Gigantesco"},
		{ID: "c12", Categoria: CategoriaSensorial, Esquerda: "Curto", Direita: "Infinito"},
		{ID: "c13", Categoria: CategoriaSensorial, Esquerda: "Cheiroso", Direita: "Fedorento"},
		{ID: "c14", Categoria: CategoriaSensorial, Esquerda: "Flexível", Direita: "Rígido"},
		{ID: "c15", Categoria: CategoriaSensorial, Esquerda: "Transparente", Direita: "Opaco"},
		{ID: "c16", Categoria: CategoriaSensorial, Esquerda: "Raso", Direita: "Profundo"},
		{ID: "c17", Categoria: CategoriaSensorial, Esquerda: "Limpo", Direita: "Imundo"},
		{ID: "c18", Categoria: CategoriaSensorial, Esquerda: "Afiado", Direita: "Cego"},
		{ID: "c19", Categoria: CategoriaSensorial, Esquerda: "Estável", Direita: "Instável"},
		{ID: "c20", Categoria: CategoriaSensorial, Esquerda: "Oco", Direita: "Maciço"},
		{ID: "c21", Categoria: CategoriaSensorial, Esquerda: "Liso", Direita: "Enrugado"},
		{ID: "c22", Categoria: CategoriaSensorial, Esquerda: "Vazio", Direita: "Lotado"},
		{ID: "c23", Categoria: CategoriaSensorial, Esquerda: "Cru", Direita: "Bem passado"},
		{ID: "c24", Categoria: CategoriaSensorial, Esquerda: "Natural", Direita: "Artificial"},
		{ID: "c25", Categoria: CategoriaSensorial, Esquerda: "Líquido", Direita: "Sólido"},
		{ID: "c26", Categoria: CategoriaSensorial, Esquerda: "Perto", Direita: "Distante"},
		{ID: "c27", Categoria: CategoriaSensorial, Esquerda: "Baixo", Direita: "Alto"},
		{ID: "c28", Categoria: CategoriaSensorial, Esquerda: "Estreito", Direita: "Largo"},
		{ID: "c29", Categoria: CategoriaSensorial, Esquerda: "Insosso", Direita: "Salgado"},
		{ID: "c30", Categoria: CategoriaSensorial, Esquerda: "Invisível", Direita: "Chamativo"},

		// =========================================================================
		// 2. Juízos de Valor e Opinião (30 espectros: c31 a c60)
		// =========================================================================
		{ID: "c31", Categoria: CategoriaJuizoValor, Esquerda: "Subestimado", Direita: "Superestimado"},
		{ID: "c32", Categoria: CategoriaJuizoValor, Esquerda: "Barato", Direita: "Luxuoso"},
		{ID: "c33", Categoria: CategoriaJuizoValor, Esquerda: "Inútil", Direita: "Indispensável"},
		{ID: "c34", Categoria: CategoriaJuizoValor, Esquerda: "Saboroso", Direita: "Intragável"},
		{ID: "c35", Categoria: CategoriaJuizoValor, Esquerda: "Brega", Direita: "Elegante"},
		{ID: "c36", Categoria: CategoriaJuizoValor, Esquerda: "Obra-prima", Direita: "Lixo completo"},
		{ID: "c37", Categoria: CategoriaJuizoValor, Esquerda: "Entediante", Direita: "Empolgante"},
		{ID: "c38", Categoria: CategoriaJuizoValor, Esquerda: "Feio", Direita: "Deslumbrante"},
		{ID: "c39", Categoria: CategoriaJuizoValor, Esquerda: "Comum", Direita: "Raro"},
		{ID: "c40", Categoria: CategoriaJuizoValor, Esquerda: "Modinha passageira", Direita: "Clássico atemporal"},
		{ID: "c41", Categoria: CategoriaJuizoValor, Esquerda: "Genérico", Direita: "Original"},
		{ID: "c42", Categoria: CategoriaJuizoValor, Esquerda: "Injustiçado", Direita: "Merecidamente odiado"},
		{ID: "c43", Categoria: CategoriaJuizoValor, Esquerda: "Bom investimento", Direita: "Dinheiro jogado fora"},
		{ID: "c44", Categoria: CategoriaJuizoValor, Esquerda: "Prazer culposo", Direita: "Orgulho de gostar"},
		{ID: "c45", Categoria: CategoriaJuizoValor, Esquerda: "Calmo", Direita: "Estressante"},
		{ID: "c46", Categoria: CategoriaJuizoValor, Esquerda: "Confiável", Direita: "Suspeito"},
		{ID: "c47", Categoria: CategoriaJuizoValor, Esquerda: "Autêntico", Direita: "Forçado"},
		{ID: "c48", Categoria: CategoriaJuizoValor, Esquerda: "Necessidade básica", Direita: "Capricho"},
		{ID: "c49", Categoria: CategoriaJuizoValor, Esquerda: "Inovador", Direita: "Ultrapassado"},
		{ID: "c50", Categoria: CategoriaJuizoValor, Esquerda: "Reconfortante", Direita: "Perturbador"},
		{ID: "c51", Categoria: CategoriaJuizoValor, Esquerda: "Marcante", Direita: "Insignificante"},
		{ID: "c52", Categoria: CategoriaJuizoValor, Esquerda: "Justo", Direita: "Injusto"},
		{ID: "c53", Categoria: CategoriaJuizoValor, Esquerda: "Engraçado", Direita: "Sem graça"},
		{ID: "c54", Categoria: CategoriaJuizoValor, Esquerda: "Saudável", Direita: "Prejudicial"},
		{ID: "c55", Categoria: CategoriaJuizoValor, Esquerda: "Sofisticado", Direita: "Rústico"},
		{ID: "c56", Categoria: CategoriaJuizoValor, Esquerda: "Inspirador", Direita: "Deprimente"},
		{ID: "c57", Categoria: CategoriaJuizoValor, Esquerda: "Sucesso", Direita: "Fracasso"},
		{ID: "c58", Categoria: CategoriaJuizoValor, Esquerda: "Santo", Direita: "Profano"},
		{ID: "c59", Categoria: CategoriaJuizoValor, Esquerda: "Presente incrível", Direita: "Presente horrível"},
		{ID: "c60", Categoria: CategoriaJuizoValor, Esquerda: "Todo mundo ama", Direita: "Todo mundo odeia"},

		// =========================================================================
		// 3. Comportamento e Cotidiano (30 espectros: c61 a c90)
		// =========================================================================
		{ID: "c61", Categoria: CategoriaComportamento, Esquerda: "Anônimo", Direita: "Famoso"},
		{ID: "c62", Categoria: CategoriaComportamento, Esquerda: "Inofensivo", Direita: "Perigoso"},
		{ID: "c63", Categoria: CategoriaComportamento, Esquerda: "Normal", Direita: "Bizarro"},
		{ID: "c64", Categoria: CategoriaComportamento, Esquerda: "Moral", Direita: "Imoral"},
		{ID: "c65", Categoria: CategoriaComportamento, Esquerda: "Triste", Direita: "Feliz"},
		{ID: "c66", Categoria: CategoriaComportamento, Esquerda: "Aceitável em público", Direita: "Vergonha alheia"},
		{ID: "c67", Categoria: CategoriaComportamento, Esquerda: "Fácil de perdoar", Direita: "Imperdoável"},
		{ID: "c68", Categoria: CategoriaComportamento, Esquerda: "Educado", Direita: "Arrogante"},
		{ID: "c69", Categoria: CategoriaComportamento, Esquerda: "Hábito saudável", Direita: "Vício"},
		{ID: "c70", Categoria: CategoriaComportamento, Esquerda: "Segredo bobo", Direita: "Segredo de Estado"},
		{ID: "c71", Categoria: CategoriaComportamento, Esquerda: "Assunto de elevador", Direita: "Conversa de madrugada"},
		{ID: "c72", Categoria: CategoriaComportamento, Esquerda: "Coisa de criança", Direita: "Coisa de idoso"},
		{ID: "c73", Categoria: CategoriaComportamento, Esquerda: "Bom primeiro encontro", Direita: "Pior primeiro encontro"},
		{ID: "c74", Categoria: CategoriaComportamento, Esquerda: "Mentira inofensiva", Direita: "Traição pesada"},
		{ID: "c75", Categoria: CategoriaComportamento, Esquerda: "Rotina diária", Direita: "Único na vida"},
		{ID: "c76", Categoria: CategoriaComportamento, Esquerda: "Impulsivo", Direita: "Calculista"},
		{ID: "c77", Categoria: CategoriaComportamento, Esquerda: "Tímido", Direita: "Extrovertido"},
		{ID: "c78", Categoria: CategoriaComportamento, Esquerda: "Bagunceiro", Direita: "Metódico"},
		{ID: "c79", Categoria: CategoriaComportamento, Esquerda: "Paciente", Direita: "Explosivo"},
		{ID: "c80", Categoria: CategoriaComportamento, Esquerda: "Egoísta", Direita: "Altruísta"},
		{ID: "c81", Categoria: CategoriaComportamento, Esquerda: "Azar passageiro", Direita: "Tragédia total"},
		{ID: "c82", Categoria: CategoriaComportamento, Esquerda: "Desculpa esfarrapada", Direita: "Motivo justo"},
		{ID: "c83", Categoria: CategoriaComportamento, Esquerda: "Elogio sincero", Direita: "Ofensa disfarçada"},
		{ID: "c84", Categoria: CategoriaComportamento, Esquerda: "Amizade casual", Direita: "Alma gêmea"},
		{ID: "c85", Categoria: CategoriaComportamento, Esquerda: "Trabalho dos sonhos", Direita: "Pesadelo corporativo"},
		{ID: "c86", Categoria: CategoriaComportamento, Esquerda: "Emergência real", Direita: "Drama exagerado"},
		{ID: "c87", Categoria: CategoriaComportamento, Esquerda: "Corajoso", Direita: "Imprudente"},
		{ID: "c88", Categoria: CategoriaComportamento, Esquerda: "Poupador", Direita: "Mão de vaca"},
		{ID: "c89", Categoria: CategoriaComportamento, Esquerda: "Comida rotineira", Direita: "Manjar dos deuses"},
		{ID: "c90", Categoria: CategoriaComportamento, Esquerda: "Silêncio confortável", Direita: "Clima tenso"},

		// =========================================================================
		// 4. Complexidade e Esforço (30 espectros: c91 a c120)
		// =========================================================================
		{ID: "c91", Categoria: CategoriaComplexidade, Esquerda: "Fácil de aprender", Direita: "Difícil de dominar"},
		{ID: "c92", Categoria: CategoriaComplexidade, Esquerda: "Tarefa de 5 minutos", Direita: "Projeto de vida"},
		{ID: "c93", Categoria: CategoriaComplexidade, Esquerda: "Exige sorte", Direita: "Exige talento"},
		{ID: "c94", Categoria: CategoriaComplexidade, Esquerda: "Intuitivo", Direita: "Confuso"},
		{ID: "c95", Categoria: CategoriaComplexidade, Esquerda: "Trabalho braçal", Direita: "Trabalho mental"},
		{ID: "c96", Categoria: CategoriaComplexidade, Esquerda: "Simples", Direita: "Complexo"},
		{ID: "c97", Categoria: CategoriaComplexidade, Esquerda: "Feito no improviso", Direita: "Perfeccionista"},
		{ID: "c98", Categoria: CategoriaComplexidade, Esquerda: "Espontâneo", Direita: "Planejado nos detalhes"},
		{ID: "c99", Categoria: CategoriaComplexidade, Esquerda: "Qualquer um faz", Direita: "Só especialista resolve"},
		{ID: "c100", Categoria: CategoriaComplexidade, Esquerda: "Preguiçoso", Direita: "Produtivo"},
		{ID: "c101", Categoria: CategoriaComplexidade, Esquerda: "Baixo risco", Direita: "Alto risco"},
		{ID: "c102", Categoria: CategoriaComplexidade, Esquerda: "Solução definitiva", Direita: "Gambiarra temporária"},
		{ID: "c103", Categoria: CategoriaComplexidade, Esquerda: "Acessível", Direita: "Inacessível"},
		{ID: "c104", Categoria: CategoriaComplexidade, Esquerda: "Teoria pura", Direita: "Prática pura"},
		{ID: "c105", Categoria: CategoriaComplexidade, Esquerda: "Automatizado", Direita: "Artesanal"},
		{ID: "c106", Categoria: CategoriaComplexidade, Esquerda: "Amador", Direita: "Profissional"},
		{ID: "c107", Categoria: CategoriaComplexidade, Esquerda: "Cansaço físico", Direita: "Exaustão mental"},
		{ID: "c108", Categoria: CategoriaComplexidade, Esquerda: "Decisão óbvia", Direita: "Dilema sem saída"},
		{ID: "c109", Categoria: CategoriaComplexidade, Esquerda: "Instrução clara", Direita: "Enigma indecifrável"},
		{ID: "c110", Categoria: CategoriaComplexidade, Esquerda: "Conserto simples", Direita: "Perda total"},
		{ID: "c111", Categoria: CategoriaComplexidade, Esquerda: "Treino leve", Direita: "Desafio extremo"},
		{ID: "c112", Categoria: CategoriaComplexidade, Esquerda: "Habilidade comum", Direita: "Dom raro"},
		{ID: "c113", Categoria: CategoriaComplexidade, Esquerda: "Passo a passo", Direita: "Caos criativo"},
		{ID: "c114", Categoria: CategoriaComplexidade, Esquerda: "Esforço individual", Direita: "Trabalho em equipe"},
		{ID: "c115", Categoria: CategoriaComplexidade, Esquerda: "Solução elegante", Direita: "Força bruta"},
		{ID: "c116", Categoria: CategoriaComplexidade, Esquerda: "Concreto", Direita: "Abstrato"},
		{ID: "c117", Categoria: CategoriaComplexidade, Esquerda: "Previsível", Direita: "Imprevisível"},
		{ID: "c118", Categoria: CategoriaComplexidade, Esquerda: "Controle total", Direita: "Fora de controle"},
		{ID: "c119", Categoria: CategoriaComplexidade, Esquerda: "Piloto automático", Direita: "Foco absoluto"},
		{ID: "c120", Categoria: CategoriaComplexidade, Esquerda: "Obstáculo pequeno", Direita: "Missão impossível"},

		// =========================================================================
		// 5. Cultura Pop e Ficcional (30 espectros: c121 a c150)
		// =========================================================================
		{ID: "c121", Categoria: CategoriaCulturaPop, Esquerda: "Antigo", Direita: "Futurista"},
		{ID: "c122", Categoria: CategoriaCulturaPop, Esquerda: "Herói puro", Direita: "Vilão absoluto"},
		{ID: "c123", Categoria: CategoriaCulturaPop, Esquerda: "Habilidade inútil", Direita: "Superpoder apelão"},
		{ID: "c124", Categoria: CategoriaCulturaPop, Esquerda: "Fofo", Direita: "Aterrorizante"},
		{ID: "c125", Categoria: CategoriaCulturaPop, Esquerda: "Realista", Direita: "Fantasioso"},
		{ID: "c126", Categoria: CategoriaCulturaPop, Esquerda: "Mundo utópico", Direita: "Cenário pós-apocalíptico"},
		{ID: "c127", Categoria: CategoriaCulturaPop, Esquerda: "Magia mística", Direita: "Tecnologia avançada"},
		{ID: "c128", Categoria: CategoriaCulturaPop, Esquerda: "Figurante descartável", Direita: "Protagonista lendário"},
		{ID: "c129", Categoria: CategoriaCulturaPop, Esquerda: "Monstro amigável", Direita: "Criatura mortal"},
		{ID: "c130", Categoria: CategoriaCulturaPop, Esquerda: "Filme de sessão da tarde", Direita: "Cinema cult"},
		{ID: "c131", Categoria: CategoriaCulturaPop, Esquerda: "Vilão incompreendido", Direita: "Pura maldade"},
		{ID: "c132", Categoria: CategoriaCulturaPop, Esquerda: "Arma improvisada", Direita: "Artefato lendário"},
		{ID: "c133", Categoria: CategoriaCulturaPop, Esquerda: "Sidequest", Direita: "Boss Battle"},
		{ID: "c134", Categoria: CategoriaCulturaPop, Esquerda: "Lugar seguro", Direita: "Covil dos lobos"},
		{ID: "c135", Categoria: CategoriaCulturaPop, Esquerda: "Clichê batido", Direita: "Reviravolta genial"},
		{ID: "c136", Categoria: CategoriaCulturaPop, Esquerda: "Poder discreto", Direita: "Poder destrutivo"},
		{ID: "c137", Categoria: CategoriaCulturaPop, Esquerda: "Aliado leal", Direita: "Traidor óbvio"},
		{ID: "c138", Categoria: CategoriaCulturaPop, Esquerda: "O Lado Puro da Força", Direita: "O Lado Negro da Força"},
		{ID: "c139", Categoria: CategoriaCulturaPop, Esquerda: "Conto de fadas", Direita: "Terror psicológico"},
		{ID: "c140", Categoria: CategoriaCulturaPop, Esquerda: "Música de fundo", Direita: "Tema épico"},
		{ID: "c141", Categoria: CategoriaCulturaPop, Esquerda: "Item comum", Direita: "Relíquia sagrada"},
		{ID: "c142", Categoria: CategoriaCulturaPop, Esquerda: "Início da jornada", Direita: "Nível máximo"},
		{ID: "c143", Categoria: CategoriaCulturaPop, Esquerda: "Planeta pacífico", Direita: "Mundo hostil"},
		{ID: "c144", Categoria: CategoriaCulturaPop, Esquerda: "Robô prestativo", Direita: "IA rebelde"},
		{ID: "c145", Categoria: CategoriaCulturaPop, Esquerda: "Disfarce perfeito", Direita: "Fantasia ridícula"},
		{ID: "c146", Categoria: CategoriaCulturaPop, Esquerda: "Viagem no tempo segura", Direita: "Paradoxo temporal"},
		{ID: "c147", Categoria: CategoriaCulturaPop, Esquerda: "Lenda urbana", Direita: "Fato histórico"},
		{ID: "c148", Categoria: CategoriaCulturaPop, Esquerda: "Comédia pastelão", Direita: "Drama emocionante"},
		{ID: "c149", Categoria: CategoriaCulturaPop, Esquerda: "Final feliz", Direita: "Final trágico"},
		{ID: "c150", Categoria: CategoriaCulturaPop, Esquerda: "Universo pacato", Direita: "Multiverso em colapso"},
	}
}
