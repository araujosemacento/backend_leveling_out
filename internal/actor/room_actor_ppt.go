package actor

import (
	"time"

	"leveling-out-backend/internal/entities"
)

// ==========================================
// Handlers do Minijogo de Iniciativa (PPT)
// ==========================================

func (r *RoomActor) handleLancePpt(playerID string, payload map[string]interface{}) {
	if r.estado.Fase != entities.FaseIniciativa || r.estado.Ppt.Estado != "ESCOLHA" {
		return
	}

	lance, _ := payload["lance"].(string)
	jogador, ok := r.estado.Jogadores[playerID]
	if !ok {
		return
	}

	ambosJogaram, err := r.pptArbiter.SubmitMove(r.codigo, playerID, jogador.Equipe, lance)
	if err != nil {
		return
	}

	switch jogador.Equipe {
	case entities.EquipePato:
		r.estado.Ppt.EnviouA = true
	case entities.EquipeGanso:
		r.estado.Ppt.EnviouB = true
	}

	if ambosJogaram {
		r.iniciarContagemPpt()
	} else {
		r.broadcastZeroTrust()
	}
}

func (r *RoomActor) iniciarContagemPpt() {
	r.estado.Ppt.Estado = "CONTAGEM"
	r.estado.Ppt.RevelacaoEm = time.Now().Add(3 * time.Second)
	r.broadcastZeroTrust()
}

// ==========================================
// Ticks e Timers Temporais Autoritativos
// ==========================================

func (r *RoomActor) processarTicksTemporais() {
	now := time.Now()

	// 0. Purga de participantes desconectados no Lobby após carência de 5 segundos.
	// Garante que recarregamentos rápidos (F5) não removam o anfitrião de seu posto
	// nem retirem o participante de sua equipe escolhida.
	if r.estado.Fase == entities.FaseLobby {
		alterou := false
		for id, j := range r.estado.Jogadores {
			if !j.IsOnline && now.Sub(j.LastSeen) >= 5*time.Second {
				delete(r.estado.Jogadores, id)
				alterou = true
				if r.estado.HostID == id {
					r.promoverNovoHost()
				}
			}
		}
		if alterou {
			r.totalJogadores.Store(int32(len(r.estado.Jogadores)))
			if len(r.estado.Jogadores) == 0 {
				r.dispositivosBanidos = make(map[string]bool)
				r.jogadoresBanidos = make(map[string]bool)
				r.estado.HostID = ""
			}
			r.broadcastZeroTrust()
		}
	}

	// 1. Timeout de 10 segundos no PPT (sorteia lance por quem não jogou)
	if r.estado.Fase == entities.FaseIniciativa && r.estado.Ppt.Estado == "ESCOLHA" {
		if now.After(r.estado.Ppt.Prazo) {
			r.pptArbiter.Preencher(r.codigo, entities.EquipePato, entities.EquipeGanso)
			r.estado.Ppt.EnviouA = true
			r.estado.Ppt.EnviouB = true
			r.iniciarContagemPpt()
			return
		}
	}

	// 2. Término da contagem regressiva 3..2..1..Já! do PPT
	if r.estado.Fase == entities.FaseIniciativa && r.estado.Ppt.Estado == "CONTAGEM" {
		if now.After(r.estado.Ppt.RevelacaoEm) {
			res, err := r.pptArbiter.Arbitrar(r.codigo)
			if err == nil {
				r.estado.Ppt.Estado = "RESULTADO"
				r.estado.Ppt.Resultado = res.Resultado
				r.estado.Ppt.LanceA = res.Lances[entities.EquipePato]
				r.estado.Ppt.LanceB = res.Lances[entities.EquipeGanso]

				if res.Resultado == entities.PptResultadoEmpate {
					r.timerTransicao = now.Add(2500 * time.Millisecond)
					r.acaoTransicao = "REINICIAR_PPT_EMPATE"
				} else {
					r.estado.EquipeAtiva = res.Vencedor
					r.timerTransicao = now.Add(4 * time.Second)
					r.acaoTransicao = "INICIAR_PRIMEIRA_RODADA"
				}
				r.broadcastZeroTrust()
				return
			}
		}
	}

	// 3. Ações automáticas agendadas
	if !r.timerTransicao.IsZero() && now.After(r.timerTransicao) {
		acao := r.acaoTransicao
		r.timerTransicao = time.Time{}
		r.acaoTransicao = ""

		switch acao {
		case "REINICIAR_PPT_EMPATE":
			repA := r.estado.Ppt.RepresentanteA
			repB := r.estado.Ppt.RepresentanteB
			r.estado.Ppt = entities.PptContestState{
				Estado:         "ESCOLHA",
				Prazo:          now.Add(10 * time.Second),
				RepresentanteA: repA,
				RepresentanteB: repB,
			}
			r.pptArbiter.Abrir(r.codigo, repA, repB)
			r.broadcastZeroTrust()

		case "INICIAR_PRIMEIRA_RODADA":
			r.prepararProximaRodada()

		case "AVANCAR_APOS_REVELACAO":
			// Verifica vitória (10 pontos)
			if r.estado.PlacarA >= entities.MetaPontuacaoPadrao || r.estado.PlacarB >= entities.MetaPontuacaoPadrao {
				r.estado.Fase = entities.FaseFimJogo
				if r.estado.PlacarA >= entities.MetaPontuacaoPadrao {
					r.estado.Vencedor = entities.EquipePato
				} else {
					r.estado.Vencedor = entities.EquipeGanso
				}
				r.estado.MotivoFim = "VITORIA"
				r.broadcastZeroTrust()
			} else {
				// Alterna equipe da vez
				if r.estado.EquipeAtiva == entities.EquipePato {
					r.estado.EquipeAtiva = entities.EquipeGanso
				} else {
					r.estado.EquipeAtiva = entities.EquipePato
				}
				r.estado.RodadaAtual++
				r.prepararProximaRodada()
			}
		}
	}
}
