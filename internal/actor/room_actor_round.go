package actor

import (
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

func (r *RoomActor) handleEscolherEspectro(playerID string, payload map[string]interface{}) {
	if r.estado.Fase != entities.FaseRodadaEspectro {
		return
	}
	if playerID != r.estado.Rodada.CodificadorID {
		return
	}

	cartaID, _ := payload["cartaId"].(string)
	carta, achou := r.spectrums.ObterPorID(cartaID)
	if !achou {
		if len(r.cartasOpcoes) > 0 {
			carta = r.cartasOpcoes[0]
		}
	}

	r.estado.Rodada.EspectroEsquerda = carta.Esquerda
	r.estado.Rodada.EspectroDireita = carta.Direita
	r.estado.Rodada.MetaOculta = r.secretTarget.Generate()
	r.estado.Rodada.IniciadaEm = time.Now()
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

func (r *RoomActor) prepararProximaRodada() {
	r.estado.Fase = entities.FaseRodadaEspectro

	// Sorteia duas cartas para o Codificador escolher
	c1, c2 := r.spectrums.SortearPar()
	r.cartasOpcoes = []components.SpectrumCard{c1, c2}

	// Define Codificador e Palpiteiro da equipe ativa
	var membrosEquipe []string
	for _, j := range r.estado.Jogadores {
		if j.Equipe == r.estado.EquipeAtiva {
			membrosEquipe = append(membrosEquipe, j.PlayerID)
		}
	}

	// Ordena por antiguidade para alternar papéis
	sort.Strings(membrosEquipe)
	codificador := ""
	palpiteiro := ""
	if len(membrosEquipe) >= 2 {
		// Alterna a cada rodada baseado no número da rodada
		if r.estado.RodadaAtual%2 == 1 {
			codificador = membrosEquipe[0]
			palpiteiro = membrosEquipe[1]
		} else {
			codificador = membrosEquipe[1]
			palpiteiro = membrosEquipe[0]
		}
	} else if len(membrosEquipe) == 1 {
		codificador = membrosEquipe[0]
		palpiteiro = membrosEquipe[0]
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
		Numero:        r.estado.RodadaAtual,
		Equipe:        r.estado.EquipeAtiva,
		CodificadorID: codificador,
		PalpiteiroID:  palpiteiro,
	}

	r.broadcastZeroTrust()
}
