package hub

import (
	"sync"
	"time"

	"leveling-out-backend/internal/actor"
)

// IntervaloLimpezaSalas define a frequência padrão de checagem de salas ociosas
const IntervaloLimpezaSalas = 5 * time.Minute

// LimiteInatividadeSalas define o tempo máximo que uma sala vazia permanece na RAM
const LimiteInatividadeSalas = 30 * time.Minute

// LobbyHub gerencia as instâncias ativas de salas em memória
type LobbyHub struct {
	mu    sync.RWMutex
	salas map[string]*actor.RoomActor
}

// NewLobbyHub inicializa o hub com rotina em background para limpeza de salas ociosas
func NewLobbyHub() *LobbyHub {
	h := &LobbyHub{
		salas: make(map[string]*actor.RoomActor),
	}
	go h.iniciarCicloLimpeza(IntervaloLimpezaSalas, LimiteInatividadeSalas)
	return h
}

// ObterOuCriar busca uma sala existente pelo código ou instancia um novo RoomActor
func (h *LobbyHub) ObterOuCriar(codigo string) *actor.RoomActor {
	h.mu.Lock()
	defer h.mu.Unlock()

	if sala, existe := h.salas[codigo]; existe {
		if !sala.EstaVazia() || time.Since(sala.UltimoAcesso()) < 10*time.Second {
			return sala
		}
		// A sala anterior ficou totalmente sem conexões e sem jogadores há mais de 10s (sessão abandonada).
		// Encerra o ator anterior para iniciar uma sessão do zero, sem banimentos residuais.
		delete(h.salas, codigo)
		sala.Encerrar()
	}

	novaSala := actor.NewRoomActor(codigo)
	h.salas[codigo] = novaSala
	return novaSala
}

// Remover descarta uma sala da memória e encerra seu ator
func (h *LobbyHub) Remover(codigo string) {
	h.mu.Lock()
	sala, existe := h.salas[codigo]
	if existe {
		delete(h.salas, codigo)
	}
	h.mu.Unlock()

	if existe && sala != nil {
		sala.Encerrar()
	}
}

// SalasAtivas retorna o total de salas em execução na RAM
func (h *LobbyHub) SalasAtivas() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.salas)
}

// ExpurgarInativas remove do mapa e encerra todas as salas sem conexões
// cuja inatividade excedeu o tempo limite estipulado.
// Retorna o total de salas expurgadas.
func (h *LobbyHub) ExpurgarInativas(limiteInatividade time.Duration) int {
	h.mu.Lock()
	var salasParaEncerrar []*actor.RoomActor

	for codigo, sala := range h.salas {
		if sala.EstaInativo(limiteInatividade) {
			delete(h.salas, codigo)
			salasParaEncerrar = append(salasParaEncerrar, sala)
		}
	}
	h.mu.Unlock()

	// Encerra os atores fora da seção crítica para não reter o mutex do hub
	for _, sala := range salasParaEncerrar {
		sala.Encerrar()
	}

	return len(salasParaEncerrar)
}

func (h *LobbyHub) iniciarCicloLimpeza(intervalo, limiteInatividade time.Duration) {
	ticker := time.NewTicker(intervalo)
	defer ticker.Stop()

	for range ticker.C {
		h.ExpurgarInativas(limiteInatividade)
	}
}
