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

func TestRoomActorEscolhaEspectroEAlternancia(t *testing.T) {
	room := NewRoomActor("sala-espectro")
	defer room.Encerrar()

	// 4 jogadores entram
	p1 := &ClientConn{PlayerID: "p1", DeviceID: "d1", Nome: "Alice", sendChan: make(chan []byte, 10)}
	p2 := &ClientConn{PlayerID: "p2", DeviceID: "d2", Nome: "Bob", sendChan: make(chan []byte, 10)}
	p3 := &ClientConn{PlayerID: "p3", DeviceID: "d3", Nome: "Carol", sendChan: make(chan []byte, 10)}
	p4 := &ClientConn{PlayerID: "p4", DeviceID: "d4", Nome: "Dave", sendChan: make(chan []byte, 10)}

	_ = room.DespacharSincrono("CONECTAR", "p1", nil, p1)
	_ = room.DespacharSincrono("CONECTAR", "p2", nil, p2)
	_ = room.DespacharSincrono("CONECTAR", "p3", nil, p3)
	_ = room.DespacharSincrono("CONECTAR", "p4", nil, p4)

	// p1 e p2 em Pato, p3 e p4 em Ganso (p1 e p3 mais antigos)
	_ = room.DespacharSincrono(protocol.TipoEscolherEquipe, "p1", map[string]interface{}{"equipe": entities.EquipePato}, nil)
	_ = room.DespacharSincrono(protocol.TipoEscolherEquipe, "p2", map[string]interface{}{"equipe": entities.EquipePato}, nil)
	_ = room.DespacharSincrono(protocol.TipoEscolherEquipe, "p3", map[string]interface{}{"equipe": entities.EquipeGanso}, nil)
	_ = room.DespacharSincrono(protocol.TipoEscolherEquipe, "p4", map[string]interface{}{"equipe": entities.EquipeGanso}, nil)

	// Força início da rodada 1 com equipe Pato ativa
	room.estado.EquipeAtiva = entities.EquipePato
	room.estado.RodadaAtual = 1
	room.prepararProximaRodada()

	if room.estado.Fase != entities.FaseRodadaEspectro {
		t.Fatalf("Esperava fase %s, obteve %s", entities.FaseRodadaEspectro, room.estado.Fase)
	}

	// Regra Canônica 1: Líder na 1ª rodada é o veterano mais antigo da equipe (p1)
	if room.estado.Rodada.LiderRodadaID != "p1" {
		t.Fatalf("Esperava líder inicial p1, obteve %s", room.estado.Rodada.LiderRodadaID)
	}

	// Regra Canônica 2: Prazo de 30 segundos estipulado
	if room.estado.Rodada.PrazoEspectro.IsZero() || time.Until(room.estado.Rodada.PrazoEspectro) < 20*time.Second {
		t.Fatalf("Prazo de espectro inválido: %v", room.estado.Rodada.PrazoEspectro)
	}

	if len(room.cartasOpcoes) != 2 {
		t.Fatalf("Esperava 2 opções de cartas, obteve %d", len(room.cartasOpcoes))
	}
	carta0 := room.cartasOpcoes[0].ID
	carta1 := room.cartasOpcoes[1].ID

	// Visibilidade Universal: Projeção para adversário p3 e espectador deve conter as 2 cartas
	snapP3 := room.ProjetarParaJogador("p3")
	if len(snapP3.CartasEspectroOpcoes) != 2 {
		t.Fatalf("Adversário p3 deveria visualizar as 2 opções em tempo real, obteve %d", len(snapP3.CartasEspectroOpcoes))
	}

	// Zero Trust: Adversário p3 e parceiro p2 tentam pré-selecionar (apenas o líder pode pré-selecionar)
	_ = room.DespacharSincrono(protocol.TipoPreSelecionarEspectro, "p3", map[string]interface{}{"cartaId": carta0}, nil)
	if room.estado.Rodada.CartaSelecionadaID != "" {
		t.Fatalf("Adversário não pode pré-selecionar carta")
	}

	_ = room.DespacharSincrono(protocol.TipoPreSelecionarEspectro, "p2", map[string]interface{}{"cartaId": carta0}, nil)
	if room.estado.Rodada.CartaSelecionadaID != "" {
		t.Fatalf("Parceiro não-líder não pode pré-selecionar carta")
	}

	// Líder p1 pré-seleciona carta0
	_ = room.DespacharSincrono(protocol.TipoPreSelecionarEspectro, "p1", map[string]interface{}{"cartaId": carta0}, nil)
	if room.estado.Rodada.CartaSelecionadaID != carta0 {
		t.Fatalf("Líder p1 deveria ter pré-selecionado %s, obteve %s", carta0, room.estado.Rodada.CartaSelecionadaID)
	}

	// Zero Trust: Líder p1 tenta votar (voto da mãozinha é do parceiro)
	_ = room.DespacharSincrono(protocol.TipoVotarEspectro, "p1", map[string]interface{}{"cartaId": carta1}, nil)
	if room.estado.Rodada.VotoParceiroCartaID != "" {
		t.Fatalf("Líder não pode emitir voto da mãozinha")
	}

	// Parceiro p2 vota na carta1 (mãozinha)
	_ = room.DespacharSincrono(protocol.TipoVotarEspectro, "p2", map[string]interface{}{"cartaId": carta1}, nil)
	if room.estado.Rodada.VotoParceiroCartaID != carta1 {
		t.Fatalf("Parceiro p2 deveria ter votado em %s, obteve %s", carta1, room.estado.Rodada.VotoParceiroCartaID)
	}

	// Toggle: Parceiro p2 vota novamente na mesma carta1 -> desmarca
	_ = room.DespacharSincrono(protocol.TipoVotarEspectro, "p2", map[string]interface{}{"cartaId": carta1}, nil)
	if room.estado.Rodada.VotoParceiroCartaID != "" {
		t.Fatalf("Voto deveria ter sido desmarcado no toggle")
	}

	// Parceiro p2 marca novamente carta1
	_ = room.DespacharSincrono(protocol.TipoVotarEspectro, "p2", map[string]interface{}{"cartaId": carta1}, nil)

	// Zero Trust: p2 (não-líder) tenta confirmar a escolha
	_ = room.DespacharSincrono(protocol.TipoEscolherEspectro, "p2", map[string]interface{}{"cartaId": carta1}, nil)
	if room.estado.Fase != entities.FaseRodadaEspectro {
		t.Fatalf("Não-líder p2 não pode confirmar a carta")
	}

	// Líder p1 confirma a carta0
	_ = room.DespacharSincrono(protocol.TipoEscolherEspectro, "p1", map[string]interface{}{"cartaId": carta0}, nil)
	if room.estado.Fase != entities.FaseRodadaDica {
		t.Fatalf("Esperava transição para %s, obteve %s", entities.FaseRodadaDica, room.estado.Fase)
	}
	if !room.estado.Rodada.PrazoEspectro.IsZero() {
		t.Fatalf("Prazo de espectro deveria ter sido zerado após confirmação")
	}

	// Grava quem foi o codificador e o líder da rodada 1 da Equipe Pato
	cod1 := room.estado.Rodada.CodificadorID
	palp1 := room.estado.Rodada.PalpiteiroID
	lider1 := room.estado.Rodada.LiderRodadaID

	if cod1 == palp1 {
		t.Fatalf("Codificador e palpiteiro não podem ser a mesma pessoa com 2 jogadores")
	}

	// Avança para a rodada seguinte da Equipe Pato (simulando rodada 3)
	room.estado.RodadaAtual = 3
	room.estado.EquipeAtiva = entities.EquipePato
	room.prepararProximaRodada()

	// Regra Canônica de Alternância Compulsória:
	// - Líder deve OBRIGATORIAMENTE alternar (se era p1, agora é p2)
	if room.estado.Rodada.LiderRodadaID == lider1 {
		t.Fatalf("Liderança não alternou! Era %s e permaneceu %s", lider1, room.estado.Rodada.LiderRodadaID)
	}
	// - Codificador deve OBRIGATORIAMENTE alternar (quem codificou agora palpita e vice-versa)
	if room.estado.Rodada.CodificadorID == cod1 {
		t.Fatalf("Codificador não alternou! Era %s e permaneceu %s", cod1, room.estado.Rodada.CodificadorID)
	}
	if room.estado.Rodada.PalpiteiroID == palp1 {
		t.Fatalf("Palpiteiro não alternou! Era %s e permaneceu %s", palp1, room.estado.Rodada.PalpiteiroID)
	}
}

func TestRoomActorTimeoutEspectro(t *testing.T) {
	room := NewRoomActor("sala-timeout-espectro")
	defer room.Encerrar()

	p1 := &ClientConn{PlayerID: "p1", Nome: "Alice", sendChan: make(chan []byte, 10)}
	p2 := &ClientConn{PlayerID: "p2", Nome: "Bob", sendChan: make(chan []byte, 10)}
	_ = room.DespacharSincrono("CONECTAR", "p1", nil, p1)
	_ = room.DespacharSincrono("CONECTAR", "p2", nil, p2)
	_ = room.DespacharSincrono(protocol.TipoEscolherEquipe, "p1", map[string]interface{}{"equipe": entities.EquipePato}, nil)
	_ = room.DespacharSincrono(protocol.TipoEscolherEquipe, "p2", map[string]interface{}{"equipe": entities.EquipePato}, nil)

	room.estado.EquipeAtiva = entities.EquipePato
	room.prepararProximaRodada()

	// Vota com o parceiro mas líder deixa o tempo expirar
	voto := room.cartasOpcoes[1].ID
	_ = room.DespacharSincrono(protocol.TipoVotarEspectro, "p2", map[string]interface{}{"cartaId": voto}, nil)

	// Simula timeout expirado
	room.resolverTimeoutEspectro()

	if room.estado.Fase != entities.FaseRodadaDica {
		t.Fatalf("Timeout deveria ter avançado a sala para %s, está em %s", entities.FaseRodadaDica, room.estado.Fase)
	}
	if room.estado.Rodada.EspectroEsquerda != room.cartasOpcoes[1].Esquerda {
		t.Fatalf("Deveria ter adotado o voto do parceiro %s", room.cartasOpcoes[1].Esquerda)
	}
}



