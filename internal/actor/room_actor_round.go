package actor

import (
	"crypto/rand"
	"math"
	"sort"
	"strings"
	"time"

	"leveling-out-backend/internal/components"
	"leveling-out-backend/internal/entities"
	"leveling-out-backend/internal/protocol"
)

// ==========================================
// Handlers de Rodada (Espectro, Dica, Palpite)
// ==========================================

func (r *RoomActor) handleIniciarPartida(playerID string) {
	if r.estado.Fase != entities.FaseLobby || playerID != r.estado.HostID {
		return
	}
	if len(r.estado.Jogadores) != 4 {
		return
	}

	// Distribui os jogadores indecisos (ALEATORIO) de forma balanceada
	r.alocarJogadoresAleatorios()

	// Seleciona o veterano mais antigo de cada equipe (menor EquipeDesde)
	repA := r.obterRepresentanteMaisAntigo(entities.EquipePato)
	repB := r.obterRepresentanteMaisAntigo(entities.EquipeGanso)
	if repA == "" || repB == "" {
		return
	}

	r.estado.Fase = entities.FaseIniciativa
	r.estado.Ppt = entities.PptContestState{
		Estado:         "ESCOLHA",
		Prazo:          time.Now().Add(20 * time.Second),
		RepresentanteA: repA,
		RepresentanteB: repB,
	}

	r.pptArbiter.Abrir(r.codigo, repA, repB)
	r.broadcastZeroTrust()
}

func (r *RoomActor) handlePreSelecionarEspectro(playerID string, payload map[string]interface{}) {
	if r.estado.Fase != entities.FaseRodadaEspectro {
		return
	}
	// Zero Trust: Apenas o líder da rodada pode pré-marcar a carta a ser confirmada
	if playerID != r.estado.Rodada.LiderRodadaID {
		return
	}

	cartaID, _ := payload["cartaId"].(string)
	valida := false
	for _, c := range r.cartasOpcoes {
		if c.ID == cartaID {
			valida = true
			break
		}
	}
	if !valida {
		return
	}

	r.estado.Rodada.CartaSelecionadaID = cartaID
	r.broadcastZeroTrust()
}

func (r *RoomActor) handleVotarEspectro(playerID string, payload map[string]interface{}) {
	if r.estado.Fase != entities.FaseRodadaEspectro {
		return
	}

	jogador, ok := r.estado.Jogadores[playerID]
	if !ok || jogador.Equipe != r.estado.EquipeAtiva {
		return
	}

	// Zero Trust: O voto/sugestão (mãozinha) é prerrogativa do parceiro ativo da equipe
	if playerID == r.estado.Rodada.LiderRodadaID {
		return
	}

	cartaID, _ := payload["cartaId"].(string)
	valida := false
	for _, c := range r.cartasOpcoes {
		if c.ID == cartaID {
			valida = true
			break
		}
	}
	if !valida {
		return
	}

	// Toggle: se clicar na mesma carta já votada, desmarca o voto
	if r.estado.Rodada.VotoParceiroCartaID == cartaID {
		r.estado.Rodada.VotoParceiroCartaID = ""
	} else {
		r.estado.Rodada.VotoParceiroCartaID = cartaID
	}

	r.broadcastZeroTrust()
}

func (r *RoomActor) handleEscolherEspectro(playerID string, payload map[string]interface{}) {
	if r.estado.Fase != entities.FaseRodadaEspectro {
		return
	}
	// Zero Trust: Apenas o líder ativo da rodada detém autoridade para confirmar
	if playerID != r.estado.Rodada.LiderRodadaID {
		return
	}

	cartaID, _ := payload["cartaId"].(string)
	if cartaID == "" {
		cartaID = r.estado.Rodada.CartaSelecionadaID
	}
	if cartaID == "" {
		cartaID = r.estado.Rodada.VotoParceiroCartaID
	}

	r.efetivarEscolhaEspectro(cartaID)
}

func (r *RoomActor) resolverTimeoutEspectro() {
	if r.estado.Fase != entities.FaseRodadaEspectro {
		return
	}

	// Fallback autoritativo por timeout:
	// 1. Carta pré-marcada pelo líder
	// 2. Voto do parceiro (mãozinha)
	// 3. Sorteio criptográfico aleatório (50% de chance para cada)
	cartaID := r.estado.Rodada.CartaSelecionadaID
	if cartaID == "" {
		cartaID = r.estado.Rodada.VotoParceiroCartaID
	}
	if cartaID == "" && len(r.cartasOpcoes) > 0 {
		b := make([]byte, 1)
		_, _ = rand.Read(b)
		idx := int(b[0]) % len(r.cartasOpcoes)
		cartaID = r.cartasOpcoes[idx].ID
	}

	r.efetivarEscolhaEspectro(cartaID)
}

func (r *RoomActor) efetivarEscolhaEspectro(cartaID string) {
	var carta components.SpectrumCard
	achou := false
	for _, c := range r.cartasOpcoes {
		if c.ID == cartaID {
			carta = c
			achou = true
			break
		}
	}

	if !achou {
		c, ok := r.spectrums.ObterPorID(cartaID)
		if ok {
			carta = c
			achou = true
		} else if len(r.cartasOpcoes) > 0 {
			carta = r.cartasOpcoes[0]
			achou = true
		}
	}

	if !achou {
		return
	}

	r.estado.Rodada.EspectroEsquerda = carta.Esquerda
	r.estado.Rodada.EspectroDireita = carta.Direita
	r.estado.Rodada.MetaOculta = r.secretTarget.Generate()
	r.estado.Rodada.IniciadaEm = time.Now()
	r.estado.Rodada.PrazoEspectro = time.Time{} // Conclui e zera o prazo
	r.estado.Fase = entities.FaseRodadaDica

	r.broadcastZeroTrust()
}

func (r *RoomActor) handleSubmeterDica(playerID string, payload map[string]interface{}) {
	if r.estado.Fase != entities.FaseRodadaDica || playerID != r.estado.Rodada.CodificadorID {
		return
	}

	dicaBruta, _ := payload["dica"].(string)
	dica := strings.TrimSpace(dicaBruta)
	if dica == "" {
		return
	}
	if len(dica) > 60 {
		dica = dica[:60]
	}

	r.estado.Rodada.Dica = dica
	r.estado.Rodada.SliderPosicao = 50.0 // Posição neutra inicial
	r.estado.Fase = entities.FaseRodadaPalpite

	r.broadcastZeroTrust()
}

func (r *RoomActor) handleSliderMover(playerID string, payload map[string]interface{}) {
	if r.estado.Fase != entities.FaseRodadaPalpite || playerID != r.estado.Rodada.PalpiteiroID {
		return
	}

	pos, ok := payload["posicao"].(float64)
	if !ok {
		return
	}
	if pos < 0 {
		pos = 0
	} else if pos > 100 {
		pos = 100
	}

	r.estado.Rodada.SliderPosicao = pos

	// Transmite streaming para os outros sockets da sala imediatamente
	msgOut := protocol.MensagemOut{
		Tipo:    protocol.TipoSliderStream,
		Payload: map[string]float64{"posicao": pos},
	}
	for id, conn := range r.conexoes {
		if id != playerID {
			conn.EnviarJSON(msgOut)
		}
	}
}

func (r *RoomActor) handleConfirmarPalpite(playerID string) {
	if r.estado.Fase != entities.FaseRodadaPalpite || playerID != r.estado.Rodada.PalpiteiroID {
		return
	}

	palpiteInt := int(math.Round(r.estado.Rodada.SliderPosicao))
	r.estado.Rodada.Palpite = palpiteInt
	r.estado.Rodada.ReveladaEm = time.Now()

	// Cálculo oficial de proximidade
	res := r.proximity.Calculate(r.estado.Rodada.MetaOculta, palpiteInt)
	r.estado.Rodada.Pontos = res.Pontos
	r.estado.Rodada.Tier = res.Tier
	r.estado.Rodada.Label = res.Label

	// Aplica pontuação à equipe da vez
	if r.estado.Rodada.Equipe == entities.EquipePato {
		r.estado.PlacarA += res.Pontos
	} else {
		r.estado.PlacarB += res.Pontos
	}

	// Registra no histórico analítico
	duracao := r.estado.Rodada.ReveladaEm.Sub(r.estado.Rodada.IniciadaEm).Seconds()
	r.estado.Historico = append(r.estado.Historico, entities.RodadaResumo{
		Numero:           r.estado.Rodada.Numero,
		Equipe:           r.estado.Rodada.Equipe,
		CodificadorID:    r.estado.Rodada.CodificadorID,
		PalpiteiroID:     r.estado.Rodada.PalpiteiroID,
		EspectroEsquerda: r.estado.Rodada.EspectroEsquerda,
		EspectroDireita:  r.estado.Rodada.EspectroDireita,
		MetaOculta:       r.estado.Rodada.MetaOculta,
		Palpite:          palpiteInt,
		Dica:             r.estado.Rodada.Dica,
		Pontos:           res.Pontos,
		Tier:             res.Tier,
		Diff:             res.Diff,
		DuracaoSegundos:  duracao,
	})

	r.estado.Fase = entities.FaseRodadaRevelada
	r.timerTransicao = time.Now().Add(4 * time.Second)
	r.acaoTransicao = "AVANCAR_APOS_REVELACAO"

	r.broadcastZeroTrust()
}

func (r *RoomActor) handleRevanche(_ string) {
	if r.estado.Fase != entities.FaseFimJogo {
		return
	}

	r.estado.PlacarA = 0
	r.estado.PlacarB = 0
	r.estado.Vencedor = ""
	r.estado.MotivoFim = ""
	r.estado.RodadaAtual = 1
	r.estado.Historico = nil
	r.ultimoCodificador = make(map[string]string)
	r.ultimoLider = make(map[string]string)

	// Reinicia com disputa de PPT entre os representantes
	repA := r.obterRepresentanteMaisAntigo(entities.EquipePato)
	repB := r.obterRepresentanteMaisAntigo(entities.EquipeGanso)
	r.estado.Fase = entities.FaseIniciativa
	r.estado.Ppt = entities.PptContestState{
		Estado:         "ESCOLHA",
		Prazo:          time.Now().Add(20 * time.Second),
		RepresentanteA: repA,
		RepresentanteB: repB,
	}

	r.pptArbiter.Abrir(r.codigo, repA, repB)
	r.broadcastZeroTrust()
}

func (r *RoomActor) obterMembrosEquipePorAntiguidade(equipe string) []string {
	var membros []*entities.Jogador
	for _, j := range r.estado.Jogadores {
		if j.Equipe == equipe {
			membros = append(membros, j)
		}
	}
	sort.Slice(membros, func(i, j int) bool {
		return membros[i].EquipeDesde.Before(membros[j].EquipeDesde)
	})
	var ids []string
	for _, m := range membros {
		ids = append(ids, m.PlayerID)
	}
	return ids
}

func (r *RoomActor) prepararProximaRodada() {
	r.estado.Fase = entities.FaseRodadaEspectro

	// Sorteia duas cartas atômicas do componente para escolha do Líder
	c1, c2 := r.spectrums.SortearPar()
	r.cartasOpcoes = []components.SpectrumCard{c1, c2}

	// Membros da equipe ativa ordenados por antiguidade de entrada no lobby
	membrosEquipe := r.obterMembrosEquipePorAntiguidade(r.estado.EquipeAtiva)

	codificador := ""
	palpiteiro := ""
	lider := ""

	if len(membrosEquipe) >= 2 {
		membro0 := membrosEquipe[0]
		membro1 := membrosEquipe[1]

		ultimoCod := r.ultimoCodificador[r.estado.EquipeAtiva]
		ultimoLid := r.ultimoLider[r.estado.EquipeAtiva]

		if ultimoCod == "" {
			// 1ª Rodada da Equipe:
			// - Líder é o mais antigo na equipe (membro 0)
			// - Cargos de Codificador e Palpiteiro sorteados com 50% de probabilidade
			lider = membro0

			b := make([]byte, 1)
			_, _ = rand.Read(b)
			if b[0]%2 == 0 {
				codificador = membro0
				palpiteiro = membro1
			} else {
				codificador = membro1
				palpiteiro = membro0
			}
		} else {
			// Rodadas Subsequentes da Equipe: Alternância Compulsória
			// - Revezamento compulsório de Codificador / Palpiteiro
			if ultimoCod == membro0 {
				codificador = membro1
				palpiteiro = membro0
			} else {
				codificador = membro0
				palpiteiro = membro1
			}

			// - Revezamento compulsório de Liderança
			if ultimoLid == membro0 {
				lider = membro1
			} else {
				lider = membro0
			}
		}

		r.ultimoCodificador[r.estado.EquipeAtiva] = codificador
		r.ultimoLider[r.estado.EquipeAtiva] = lider
	} else if len(membrosEquipe) == 1 {
		codificador = membrosEquipe[0]
		palpiteiro = membrosEquipe[0]
		lider = membrosEquipe[0]
		r.ultimoCodificador[r.estado.EquipeAtiva] = codificador
		r.ultimoLider[r.estado.EquipeAtiva] = lider
	}

	for _, j := range r.estado.Jogadores {
		switch j.PlayerID {
		case codificador:
			j.Papel = entities.PapelCodificador
		case palpiteiro:
			j.Papel = entities.PapelPalpiteiro
		default:
			j.Papel = entities.PapelEspectador
		}
	}

	r.estado.Rodada = entities.RodadaAtiva{
		Numero:              r.estado.RodadaAtual,
		Equipe:              r.estado.EquipeAtiva,
		CodificadorID:       codificador,
		PalpiteiroID:        palpiteiro,
		LiderRodadaID:       lider,
		CartaSelecionadaID:  "",
		VotoParceiroCartaID: "",
		PrazoEspectro:       time.Now().Add(30 * time.Second),
	}

	r.broadcastZeroTrust()
}
