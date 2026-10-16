package actor

import (
	"sort"

	"leveling-out-backend/internal/entities"
	"leveling-out-backend/internal/protocol"
)

// ==========================================
// Projeções Puras e Seguras (Zero Trust)
// ==========================================

func (r *RoomActor) broadcastZeroTrust() {
	// Lista pública de jogadores
	var listaJogadores []protocol.SnapshotJogador
	for _, j := range r.estado.Jogadores {
		listaJogadores = append(listaJogadores, protocol.SnapshotJogador{
			ID:          j.ID,
			PlayerID:    j.PlayerID,
			Nome:        j.Nome,
			Equipe:      j.Equipe,
			Papel:       j.Papel,
			IsHost:      j.IsHost,
			IsOnline:    j.IsOnline,
			EquipeDesde: j.EquipeDesde,
		})
	}

	// Ordena jogadores por nome para determinismo
	sort.Slice(listaJogadores, func(i, j int) bool {
		return listaJogadores[i].Nome < listaJogadores[j].Nome
	})

	for playerID, conn := range r.conexoes {
		proj := r.gerarProjecaoZeroTrust(playerID, listaJogadores)
		conn.EnviarJSON(protocol.MensagemOut{
			Tipo:    protocol.TipoSync,
			Payload: proj,
		})
	}
}

func (r *RoomActor) gerarProjecaoZeroTrust(playerID string, listaJogadores []protocol.SnapshotJogador) protocol.SnapshotSalaProjetado {
	jogador := r.estado.Jogadores[playerID]
	minhaEquipe := ""
	if jogador != nil {
		minhaEquipe = jogador.Equipe
	}

	// 1. Projeção de PPT (Zero Trust nos lances antes do Já!)
	snapPpt := protocol.SnapshotPpt{
		Estado:         r.estado.Ppt.Estado,
		EnviouA:        r.estado.Ppt.EnviouA,
		EnviouB:        r.estado.Ppt.EnviouB,
		RepresentanteA: r.estado.Ppt.RepresentanteA,
		RepresentanteB: r.estado.Ppt.RepresentanteB,
		Resultado:      r.estado.Ppt.Resultado,
	}
	if !r.estado.Ppt.Prazo.IsZero() {
		snapPpt.Prazo = &r.estado.Ppt.Prazo
	}
	if !r.estado.Ppt.RevelacaoEm.IsZero() {
		snapPpt.RevelacaoEm = &r.estado.Ppt.RevelacaoEm
	}

	if r.estado.Ppt.Estado == "RESULTADO" {
		snapPpt.LanceA = r.estado.Ppt.LanceA
		snapPpt.LanceB = r.estado.Ppt.LanceB
	} else if minhaEquipe == entities.EquipePato || minhaEquipe == entities.EquipeGanso {
		// Representante e companheiro de equipe veem seu próprio lance submetido em sigilo
		lancePrivado := r.pptArbiter.LanceDaEquipe(r.codigo, minhaEquipe)
		snapPpt.MeuLance = lancePrivado
	}

	// 2. Projeção da Rodada (Blindagem de Meta Oculta)
	snapRodada := protocol.SnapshotRodada{
		Numero:           r.estado.Rodada.Numero,
		Equipe:           r.estado.Rodada.Equipe,
		CodificadorID:    r.estado.Rodada.CodificadorID,
		PalpiteiroID:     r.estado.Rodada.PalpiteiroID,
		EspectroEsquerda: r.estado.Rodada.EspectroEsquerda,
		EspectroDireita:  r.estado.Rodada.EspectroDireita,
		Dica:             r.estado.Rodada.Dica,
		Pontos:           r.estado.Rodada.Pontos,
		Tier:             r.estado.Rodada.Tier,
		Label:            r.estado.Rodada.Label,
	}

	if r.estado.Fase == entities.FaseRodadaRevelada || r.estado.Fase == entities.FaseFimJogo {
		snapRodada.MetaOculta = &r.estado.Rodada.MetaOculta
		palpiteVal := r.estado.Rodada.Palpite
		snapRodada.Palpite = &palpiteVal
	} else if playerID == r.estado.Rodada.CodificadorID {
		// Apenas o Codificador da vez visualiza o tubo cheio durante a dica e palpite
		snapRodada.MetaOculta = &r.estado.Rodada.MetaOculta
	}

	// 3. Opções de espectro (visíveis para o Codificador na fase de escolha)
	var opcoes []interface{}
	if r.estado.Fase == entities.FaseRodadaEspectro && playerID == r.estado.Rodada.CodificadorID {
		for _, c := range r.cartasOpcoes {
			opcoes = append(opcoes, c)
		}
	}

	return protocol.SnapshotSalaProjetado{
		Codigo:               r.codigo,
		Fase:                 r.estado.Fase,
		Rodada:               r.estado.RodadaAtual,
		EquipeAtiva:          r.estado.EquipeAtiva,
		PlacarA:              r.estado.PlacarA,
		PlacarB:              r.estado.PlacarB,
		Vencedor:             r.estado.Vencedor,
		HostID:               r.estado.HostID,
		MotivoFim:            r.estado.MotivoFim,
		Jogadores:            listaJogadores,
		Ppt:                  snapPpt,
		RodadaAtiva:          snapRodada,
		CartasEspectroOpcoes: opcoes,
	}
}
