package actor

import (
	"sort"
	"time"

	"github.com/coder/websocket"

	"leveling-out-backend/internal/entities"
	"leveling-out-backend/internal/protocol"
)

// ==========================================
// Handlers de Conexão e Presença
// ==========================================

func (r *RoomActor) handleConectar(conn *ClientConn) {
	// 1. Checagem autoritativa de banimento por dispositivo ou player_id
	if (conn.DeviceID != "" && r.dispositivosBanidos[conn.DeviceID]) || r.jogadoresBanidos[conn.PlayerID] {
		conn.EnviarJSONESair(protocol.MensagemOut{
			Tipo:    protocol.TipoBanido,
			Payload: map[string]string{"motivo": "Você foi banido desta sala pelo anfitrião e não poderá retornar deste dispositivo."},
		}, websocket.StatusPolicyViolation, "banido_pelo_anfitriao")
		return
	}

	jogador, existe := r.estado.Jogadores[conn.PlayerID]

	if !existe {
		// Valida capacidade da sala (máx 4 jogadores no Lobby)
		if len(r.estado.Jogadores) >= 4 && r.estado.Fase == entities.FaseLobby {
			conn.EnviarJSON(protocol.MensagemOut{
				Tipo:    protocol.TipoErro,
				Payload: map[string]string{"mensagem": "A sala atingiu a lotação máxima de 4 jogadores."},
			})
			conn.Fechar()
			return
		}

		// Se a partida já estiver em andamento, não admite novos jogadores
		if r.estado.Fase != entities.FaseLobby {
			conn.EnviarJSON(protocol.MensagemOut{
				Tipo:    protocol.TipoErro,
				Payload: map[string]string{"mensagem": "A partida já está em andamento nesta sala."},
			})
			conn.Fechar()
			return
		}

		// Criação de assento para o novo jogador
		token := gerarTokenSeguro()
		conn.SessionToken = token
		isHost := len(r.estado.Jogadores) == 0

		jogador = &entities.Jogador{
			ID:           "j_" + gerarIDCurto(),
			SalaCodigo:   r.codigo,
			PlayerID:     conn.PlayerID,
			Nome:         conn.Nome,
			Equipe:       entities.EquipeAleatorio,
			Papel:        entities.PapelEspectador,
			IsHost:       isHost,
			IsOnline:     true,
			SessionToken: token,
			EquipeDesde:  time.Now(),
			LastSeen:     time.Now(),
		}

		r.estado.Jogadores[conn.PlayerID] = jogador
		if isHost {
			r.estado.HostID = conn.PlayerID
		}
		r.totalJogadores.Store(int32(len(r.estado.Jogadores)))
	} else {
		// Validação de token em reconexão
		if conn.SessionToken != "" && conn.SessionToken != jogador.SessionToken {
			conn.EnviarJSON(protocol.MensagemOut{
				Tipo:    protocol.TipoErro,
				Payload: map[string]string{"mensagem": "Token de sessão inválido para este participante."},
			})
			conn.Fechar()
			return
		}

		jogador.IsOnline = true
		jogador.LastSeen = time.Now()
		if conn.Nome != "" {
			jogador.Nome = conn.Nome
		}
	}

	// Registra socket ativo
	r.conexoes[conn.PlayerID] = conn
	r.conexoesAtivas.Store(int32(len(r.conexoes)))

	// Envia mensagem de boas-vindas com o token para persistência no cliente
	conn.EnviarJSON(protocol.MensagemOut{
		Tipo: protocol.TipoBemVindo,
		Payload: protocol.MsgBemVindo{
			PlayerID:     jogador.PlayerID,
			SessionToken: jogador.SessionToken,
			IsHost:       jogador.IsHost,
			Nome:         jogador.Nome,
			SalaCodigo:   r.codigo,
		},
	})

	r.broadcastZeroTrust()
}

func (r *RoomActor) handleDesconectar(playerID string) {
	delete(r.conexoes, playerID)
	r.conexoesAtivas.Store(int32(len(r.conexoes)))

	// Marca o participante como offline sem revogar assento de imediato.
	// Permite carência de alguns segundos para que recarregamento de página (F5)
	// não retire o anfitrião de sua posição nem remova o convidado de sua equipe.
	if j, ok := r.estado.Jogadores[playerID]; ok {
		j.IsOnline = false
		j.LastSeen = time.Now()
	}

	r.broadcastZeroTrust()
}

// ==========================================
// Handlers de Escolha de Equipe e Moderação
// ==========================================

func (r *RoomActor) handleEscolherEquipe(playerID string, payload map[string]interface{}) {
	if r.estado.Fase != entities.FaseLobby {
		return
	}
	jogador, ok := r.estado.Jogadores[playerID]
	if !ok {
		return
	}

	novaEquipe, _ := payload["equipe"].(string)
	if novaEquipe != entities.EquipePato && novaEquipe != entities.EquipeGanso && novaEquipe != entities.EquipeAleatorio {
		return
	}

	if novaEquipe == jogador.Equipe {
		return
	}

	// Valida lotação de no máximo 2 por equipe
	if novaEquipe == entities.EquipePato || novaEquipe == entities.EquipeGanso {
		contagem := 0
		for _, j := range r.estado.Jogadores {
			if j.Equipe == novaEquipe {
				contagem++
			}
		}
		if contagem >= 2 {
			return
		}
	}

	jogador.Equipe = novaEquipe
	jogador.EquipeDesde = time.Now()
	r.broadcastZeroTrust()
}

func (r *RoomActor) handleExpulsarJogador(playerID string, payload map[string]interface{}) {
	if r.estado.Fase != entities.FaseLobby || playerID != r.estado.HostID {
		return
	}

	targetID, _ := payload["targetPlayerId"].(string)
	if targetID == "" {
		targetID, _ = payload["alvoId"].(string)
	}
	if targetID == "" || targetID == playerID {
		return
	}

	if conn, ok := r.conexoes[targetID]; ok {
		conn.EnviarJSONESair(protocol.MensagemOut{
			Tipo:    protocol.TipoKickado,
			Payload: map[string]string{"motivo": "Você foi expulso da sala pelo anfitrião."},
		}, websocket.StatusCode(protocol.StatusKickado), "expulso_pelo_anfitriao")
		delete(r.conexoes, targetID)
		r.conexoesAtivas.Store(int32(len(r.conexoes)))
	}

	delete(r.estado.Jogadores, targetID)
	r.totalJogadores.Store(int32(len(r.estado.Jogadores)))
	r.broadcastZeroTrust()
}

func (r *RoomActor) handleBanirJogador(playerID string, payload map[string]interface{}) {
	if r.estado.Fase != entities.FaseLobby || playerID != r.estado.HostID {
		return
	}

	targetID, _ := payload["targetPlayerId"].(string)
	if targetID == "" {
		targetID, _ = payload["alvoId"].(string)
	}
	if targetID == "" || targetID == playerID {
		return
	}

	r.jogadoresBanidos[targetID] = true

	if conn, ok := r.conexoes[targetID]; ok {
		if conn.DeviceID != "" {
			r.dispositivosBanidos[conn.DeviceID] = true
		}
		conn.EnviarJSONESair(protocol.MensagemOut{
			Tipo:    protocol.TipoBanido,
			Payload: map[string]string{"motivo": "Você foi banido desta sala pelo anfitrião e não poderá retornar deste dispositivo."},
		}, websocket.StatusPolicyViolation, "banido_pelo_anfitriao")
		delete(r.conexoes, targetID)
		r.conexoesAtivas.Store(int32(len(r.conexoes)))
	}

	delete(r.estado.Jogadores, targetID)
	r.totalJogadores.Store(int32(len(r.estado.Jogadores)))
	r.broadcastZeroTrust()
}

func (r *RoomActor) handleDesfazerPartida(playerID string) {
	if playerID != r.estado.HostID {
		return
	}

	r.estado.Fase = entities.FaseLobby
	r.estado.PlacarA = 0
	r.estado.PlacarB = 0
	r.estado.Vencedor = ""
	r.estado.Ppt = entities.PptContestState{}
	r.estado.Rodada = entities.RodadaAtiva{}
	r.pptArbiter.Fechar(r.codigo)
	r.timerTransicao = time.Time{}
	r.acaoTransicao = ""

	for _, j := range r.estado.Jogadores {
		j.Papel = entities.PapelEspectador
	}

	r.broadcastZeroTrust()
}

func (r *RoomActor) alocarJogadoresAleatorios() {
	var indecisos []*entities.Jogador
	patoCount := 0
	gansoCount := 0

	for _, j := range r.estado.Jogadores {
		switch j.Equipe {
		case entities.EquipePato:
			patoCount++
		case entities.EquipeGanso:
			gansoCount++
		default:
			indecisos = append(indecisos, j)
		}
	}

	for _, j := range indecisos {
		if patoCount < 2 {
			j.Equipe = entities.EquipePato
			j.EquipeDesde = time.Now()
			patoCount++
		} else {
			j.Equipe = entities.EquipeGanso
			j.EquipeDesde = time.Now()
			gansoCount++
		}
	}
}

func (r *RoomActor) obterRepresentanteMaisAntigo(equipe string) string {
	var membros []*entities.Jogador
	for _, j := range r.estado.Jogadores {
		if j.Equipe == equipe {
			membros = append(membros, j)
		}
	}
	if len(membros) == 0 {
		return ""
	}
	sort.Slice(membros, func(i, j int) bool {
		return membros[i].EquipeDesde.Before(membros[j].EquipeDesde)
	})
	return membros[0].PlayerID
}

func (r *RoomActor) promoverNovoHost() {
	r.estado.HostID = ""
	if len(r.estado.Jogadores) == 0 {
		return
	}

	// Ordena jogadores por data de ingresso (mais antigo primeiro)
	var candidatos []*entities.Jogador
	for _, j := range r.estado.Jogadores {
		candidatos = append(candidatos, j)
	}

	sort.Slice(candidatos, func(i, j int) bool {
		return candidatos[i].EquipeDesde.Before(candidatos[j].EquipeDesde)
	})

	if len(candidatos) > 0 {
		novoHost := candidatos[0]
		novoHost.IsHost = true
		r.estado.HostID = novoHost.PlayerID
		for _, j := range r.estado.Jogadores {
			if j.PlayerID != novoHost.PlayerID {
				j.IsHost = false
			}
		}
	}
}

