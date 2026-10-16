package actor

import (
	"testing"
	"time"

	"leveling-out-backend/internal/components"
	"leveling-out-backend/internal/entities"
	"leveling-out-backend/internal/protocol"
)

func TestRoomActorLobbyFlow(t *testing.T) {
	room := NewRoomActor("sala-teste")
	defer room.Encerrar()

	// Simula 4 jogadores conectando
	p1 := &ClientConn{PlayerID: "p1", Nome: "Alpha", sendChan: make(chan []byte, 10)}
	p2 := &ClientConn{PlayerID: "p2", Nome: "Beta", sendChan: make(chan []byte, 10)}
	p3 := &ClientConn{PlayerID: "p3", Nome: "Gama", sendChan: make(chan []byte, 10)}
	p4 := &ClientConn{PlayerID: "p4", Nome: "Omega", sendChan: make(chan []byte, 10)}

	_ = room.DespacharSincrono("CONECTAR", "p1", nil, p1)
	_ = room.DespacharSincrono("CONECTAR", "p2", nil, p2)
	_ = room.DespacharSincrono("CONECTAR", "p3", nil, p3)
	_ = room.DespacharSincrono("CONECTAR", "p4", nil, p4)

	if len(room.estado.Jogadores) != 4 {
		t.Fatalf("Esperava 4 jogadores na sala, obteve %d", len(room.estado.Jogadores))
	}

	if room.estado.HostID != "p1" {
		t.Fatalf("Esperava que p1 fosse o host, obteve %s", room.estado.HostID)
	}

	// p1 escolhe Pato (A)
	_ = room.DespacharSincrono(protocol.TipoEscolherEquipe, "p1", map[string]interface{}{"equipe": entities.EquipePato}, nil)
	// p2 escolhe Pato (A)
	_ = room.DespacharSincrono(protocol.TipoEscolherEquipe, "p2", map[string]interface{}{"equipe": entities.EquipePato}, nil)
	// p3 tenta escolher Pato (deve ser rejeitado pois lotou com 2)
	_ = room.DespacharSincrono(protocol.TipoEscolherEquipe, "p3", map[string]interface{}{"equipe": entities.EquipePato}, nil)
	// p3 escolhe Ganso (B)
	_ = room.DespacharSincrono(protocol.TipoEscolherEquipe, "p3", map[string]interface{}{"equipe": entities.EquipeGanso}, nil)

	if room.estado.Jogadores["p3"].Equipe != entities.EquipeGanso {
		t.Fatalf("p3 deveria estar na Equipe Ganso, está em %s", room.estado.Jogadores["p3"].Equipe)
	}

	// Host inicia a partida (p4 que estava em ALEATORIO deve ir para Ganso)
	_ = room.DespacharSincrono(protocol.TipoIniciarPartida, "p1", nil, nil)

	if room.estado.Fase != entities.FaseIniciativa {
		t.Fatalf("Fase deveria ter avançado para INICIATIVA, está em %s", room.estado.Fase)
	}

	if room.estado.Jogadores["p4"].Equipe != entities.EquipeGanso {
		t.Fatalf("p4 deveria ter sido alocado automaticamente em Ganso, está em %s", room.estado.Jogadores["p4"].Equipe)
	}

	// Veteranos mais antigos devem ser os representantes (p1 em Pato e p3 em Ganso)
	if room.estado.Ppt.RepresentanteA != "p1" {
		t.Fatalf("Representante Pato esperado p1, obteve %s", room.estado.Ppt.RepresentanteA)
	}
	if room.estado.Ppt.RepresentanteB != "p3" {
		t.Fatalf("Representante Ganso esperado p3, obteve %s", room.estado.Ppt.RepresentanteB)
	}
}

func TestZeroTrustProjection(t *testing.T) {
	room := &RoomActor{
		codigo: "sala-segredo",
		estado: entities.SalaEstado{
			Codigo:      "sala-segredo",
			Fase:        entities.FaseRodadaDica,
			RodadaAtual: 1,
			Jogadores: map[string]*entities.Jogador{
				"p1": {PlayerID: "p1", Nome: "Codificador", Equipe: entities.EquipePato},
				"p2": {PlayerID: "p2", Nome: "Palpiteiro", Equipe: entities.EquipePato},
			},
			Rodada: entities.RodadaAtiva{
				Numero:        1,
				Equipe:        entities.EquipePato,
				CodificadorID: "p1",
				PalpiteiroID:  "p2",
				MetaOculta:    73,
				Dica:          "Café quente",
			},
		},
		conexoes:   make(map[string]*ClientConn),
		pptArbiter: components.NewPptArbiterComponent(),
	}

	// Projeção para p1 (Codificador)
	projCodificador := room.gerarProjecaoZeroTrust("p1", nil)
	if projCodificador.RodadaAtiva.MetaOculta == nil || *projCodificador.RodadaAtiva.MetaOculta != 73 {
		t.Fatalf("Codificador deveria ver a meta oculta 73, obteve %v", projCodificador.RodadaAtiva.MetaOculta)
	}

	// Projeção para p2 (Palpiteiro)
	projPalpiteiro := room.gerarProjecaoZeroTrust("p2", nil)
	if projPalpiteiro.RodadaAtiva.MetaOculta != nil {
		t.Fatalf("Palpiteiro NÃO pode ver a meta oculta antes da revelação! Obteve: %v", *projPalpiteiro.RodadaAtiva.MetaOculta)
	}
}

func TestRoomActorLobbyDisconnection(t *testing.T) {
	room := NewRoomActor("sala-desconexao")
	defer room.Encerrar()

	p1 := &ClientConn{PlayerID: "p1", Nome: "Host", sendChan: make(chan []byte, 10)}
	p2 := &ClientConn{PlayerID: "p2", Nome: "Convidado", sendChan: make(chan []byte, 10)}

	_ = room.DespacharSincrono("CONECTAR", "p1", nil, p1)
	_ = room.DespacharSincrono("CONECTAR", "p2", nil, p2)

	if len(room.estado.Jogadores) != 2 {
		t.Fatalf("Esperava 2 jogadores, obteve %d", len(room.estado.Jogadores))
	}

	// Convidado p2 recarrega a página ou desconecta no Lobby
	_ = room.DespacharSincrono("DESCONECTAR", "p2", nil, nil)

	// No período de carência imediata (< 5s), o jogador é marcado como offline, mas mantém o assento reservado
	if j2, existe := room.estado.Jogadores["p2"]; !existe || j2.IsOnline {
		t.Fatalf("Esperava que p2 continuasse cadastrado com IsOnline = false durante carência de reload")
	}

	// Host p1 desconecta para simular recarregamento de página (F5)
	_ = room.DespacharSincrono("DESCONECTAR", "p1", nil, nil)

	// Durante a carência (< 5s), o anfitrião NÃO perde seu privilégio de host
	if room.estado.HostID != "p1" {
		t.Fatalf("Host p1 não deveria perder cargo de anfitrião imediatamente na desconexão, obteve %s", room.estado.HostID)
	}

	// p1 recarrega a página (F5) e reconecta com sucesso
	_ = room.DespacharSincrono("CONECTAR", "p1", nil, p1)
	if room.estado.HostID != "p1" || !room.estado.Jogadores["p1"].IsOnline || !room.estado.Jogadores["p1"].IsHost {
		t.Fatalf("p1 deveria retornar como Host online após recarregar a página")
	}

	// Agora simula p2 ficando offline por mais de 5 segundos (abandono da sala)
	room.estado.Jogadores["p2"].LastSeen = time.Now().Add(-6 * time.Second)
	room.processarTicksTemporais()

	// Após a carência, p2 deve ser purgado da sala
	if _, existe := room.estado.Jogadores["p2"]; existe {
		t.Fatalf("p2 deveria ter sido purgado da sala após expiração do período de carência de 5s")
	}
}

func TestRoomActorLobbyKick(t *testing.T) {
	room := NewRoomActor("sala-kick")
	defer room.Encerrar()

	p1 := &ClientConn{PlayerID: "p1", Nome: "Host", sendChan: make(chan []byte, 10)}
	p2 := &ClientConn{PlayerID: "p2", Nome: "Bagunceiro", sendChan: make(chan []byte, 10)}

	_ = room.DespacharSincrono("CONECTAR", "p1", nil, p1)
	_ = room.DespacharSincrono("CONECTAR", "p2", nil, p2)

	// Host expulsa p2
	_ = room.DespacharSincrono(protocol.TipoExpulsarJogador, "p1", map[string]interface{}{"targetPlayerId": "p2"}, nil)

	if len(room.estado.Jogadores) != 1 {
		t.Fatalf("Esperava 1 jogador restante após kick, obteve %d", len(room.estado.Jogadores))
	}
	if _, existe := room.estado.Jogadores["p2"]; existe {
		t.Fatalf("Jogador expulso p2 não deveria mais constar na sala")
	}
}

func TestRoomActorLobbyBan(t *testing.T) {
	room := NewRoomActor("sala-ban")
	defer room.Encerrar()

	p1 := &ClientConn{PlayerID: "p1", DeviceID: "dev-host", Nome: "Host", sendChan: make(chan []byte, 10)}
	p2 := &ClientConn{PlayerID: "p2", DeviceID: "dev-infrator", Nome: "Infrator", sendChan: make(chan []byte, 10)}

	_ = room.DespacharSincrono("CONECTAR", "p1", nil, p1)
	_ = room.DespacharSincrono("CONECTAR", "p2", nil, p2)

	// Host bane p2
	_ = room.DespacharSincrono(protocol.TipoBanirJogador, "p1", map[string]interface{}{"targetPlayerId": "p2"}, nil)

	if _, existe := room.estado.Jogadores["p2"]; existe {
		t.Fatalf("Jogador banido p2 não deveria mais constar na sala")
	}

	// Tentativa 1: p2 tenta reconectar com o mesmo playerID
	p2Retry := &ClientConn{PlayerID: "p2", DeviceID: "outro-dev", Nome: "Infrator", sendChan: make(chan []byte, 10)}
	_ = room.DespacharSincrono("CONECTAR", "p2", nil, p2Retry)
	if _, existe := room.estado.Jogadores["p2"]; existe {
		t.Fatalf("Jogador banido por playerID não deveria conseguir ingressar")
	}

	// Tentativa 2: p2 tenta burlar usando novo playerID mas o mesmo deviceID
	p2Bypass := &ClientConn{PlayerID: "p2_novo", DeviceID: "dev-infrator", Nome: "Disfarcado", sendChan: make(chan []byte, 10)}
	_ = room.DespacharSincrono("CONECTAR", "p2_novo", nil, p2Bypass)
	if _, existe := room.estado.Jogadores["p2_novo"]; existe {
		t.Fatalf("Jogador banido por deviceID não deveria conseguir ingressar mesmo com novo playerID")
	}

	// Jogador legítimo p3 entra com sucesso
	p3 := &ClientConn{PlayerID: "p3", DeviceID: "dev-amigo", Nome: "Amigo", sendChan: make(chan []byte, 10)}
	_ = room.DespacharSincrono("CONECTAR", "p3", nil, p3)
	if _, existe := room.estado.Jogadores["p3"]; !existe {
		t.Fatalf("Jogador legítimo p3 deveria conseguir ingressar normalmente")
	}
}


