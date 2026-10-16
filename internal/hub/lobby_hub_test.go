package hub

import (
	"testing"
	"time"

	"leveling-out-backend/internal/actor"
)

func TestLobbyHubBasics(t *testing.T) {
	h := &LobbyHub{
		salas: make(map[string]*actor.RoomActor),
	}

	if h.SalasAtivas() != 0 {
		t.Fatalf("Esperava 0 salas ativas, obteve %d", h.SalasAtivas())
	}

	sala1 := h.ObterOuCriar("sala-1")
	if sala1 == nil {
		t.Fatal("Esperava sala instanciada, obteve nil")
	}

	if h.SalasAtivas() != 1 {
		t.Fatalf("Esperava 1 sala ativa, obteve %d", h.SalasAtivas())
	}

	// Obter a mesma sala deve retornar a mesma instância
	sala1Repetida := h.ObterOuCriar("sala-1")
	if sala1Repetida != sala1 {
		t.Fatal("Esperava a mesma instância de sala para a mesma chave")
	}

	// Remover a sala deve limpar do mapa e encerrar o ator
	h.Remover("sala-1")
	if h.SalasAtivas() != 0 {
		t.Fatalf("Esperava 0 salas após remoção, obteve %d", h.SalasAtivas())
	}
}

func TestLobbyHubExpurgarInativas(t *testing.T) {
	h := &LobbyHub{
		salas: make(map[string]*actor.RoomActor),
	}

	salaNova := h.ObterOuCriar("sala-recente")
	if salaNova == nil {
		t.Fatal("Falha ao criar sala")
	}

	// Sala acabou de ser criada, não deve ser expurgada com limite de 10 minutos
	removidas := h.ExpurgarInativas(10 * time.Minute)
	if removidas != 0 {
		t.Fatalf("Esperava 0 salas expurgadas, obteve %d", removidas)
	}
	if h.SalasAtivas() != 1 {
		t.Fatalf("Sala recente deveria continuar ativa, obteve %d", h.SalasAtivas())
	}

	// Com limite 0s (qualquer sala sem conexões ativas), ela deve ser expurgada
	time.Sleep(10 * time.Millisecond)
	removidas = h.ExpurgarInativas(1 * time.Millisecond)
	if removidas != 1 {
		t.Fatalf("Esperava 1 sala expurgada, obteve %d", removidas)
	}
	if h.SalasAtivas() != 0 {
		t.Fatalf("Esperava 0 salas após expurgo, obteve %d", h.SalasAtivas())
	}
}
