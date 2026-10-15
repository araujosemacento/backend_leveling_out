package components

import (
	"errors"
	"fmt"
	"sync"

	"leveling-out-backend/internal/entities"
)

// RodadaPptMemoria armazena os lances submetidos para uma rodada específica de uma sala
type RodadaPptMemoria struct {
	Lances    map[string]entities.PptMoveRequest // mapeado por PlayerID
	Resultado *entities.PptResultResponse
}

// PptArbiterComponent modela o ArbitroPptComponent com gerenciamento concorrente thread-safe
type PptArbiterComponent struct {
	mu      sync.RWMutex
	rodadas map[string]*RodadaPptMemoria // chave: "ppt_{salaCodigo}_{rodada}"
}

// NewPptArbiterComponent inicializa o árbitro de PPT em memória
func NewPptArbiterComponent() *PptArbiterComponent {
	return &PptArbiterComponent{
		rodadas: make(map[string]*RodadaPptMemoria),
	}
}

func (a *PptArbiterComponent) makeKey(salaCodigo string, rodada int) string {
	return fmt.Sprintf("ppt_%s_%d", salaCodigo, rodada)
}

// SubmitMove processa um lance de PPT de um participante
// Retorna o resultado, uma flag indicando se a rodada acabou de ser resolvida neste lance, e erro se houver
func (a *PptArbiterComponent) SubmitMove(req entities.PptMoveRequest) (*entities.PptResultResponse, bool, error) {
	if req.SalaCodigo == "" || req.PlayerID == "" || req.Equipe == "" || req.Lance == "" {
		return nil, false, errors.New("parâmetros incompletos para o lance de PPT")
	}

	key := a.makeKey(req.SalaCodigo, req.Rodada)

	a.mu.Lock()
	defer a.mu.Unlock()

	mem, exists := a.rodadas[key]
	if !exists {
		mem = &RodadaPptMemoria{
			Lances: make(map[string]entities.PptMoveRequest),
		}
		a.rodadas[key] = mem
	}

	// Se a rodada já tiver sido resolvida anteriormente, retorna o resultado cacheado
	if mem.Resultado != nil {
		return mem.Resultado, false, nil
	}

	// Registra o lance do jogador
	mem.Lances[req.PlayerID] = req

	// Se ainda não temos lances suficientes de ambas as equipes
	if len(mem.Lances) < 2 {
		return &entities.PptResultResponse{
			Status: entities.StatusAguardandoOponente,
			Rodada: req.Rodada,
		}, false, nil
	}

	// Extrai lances dos dois primeiros jogadores participantes
	var p1, p2 entities.PptMoveRequest
	first := true
	for _, lance := range mem.Lances {
		if first {
			p1 = lance
			first = false
		} else {
			p2 = lance
			break
		}
	}

	lanceEquipeA := p2.Lance
	if p1.Equipe == entities.EquipePato {
		lanceEquipeA = p1.Lance
	}
	lanceEquipeB := p2.Lance
	if p1.Equipe == entities.EquipeGanso {
		lanceEquipeB = p1.Lance
	}

	// Arbitragem oficial de Pedra, Papel e Tesoura
	vencedor := entities.VencedorEmpate
	resultadoSlug := "EMPATE"

	if lanceEquipeA != lanceEquipeB {
		if (lanceEquipeA == entities.LancePedra && lanceEquipeB == entities.LanceTesoura) ||
			(lanceEquipeA == entities.LancePapel && lanceEquipeB == entities.LancePedra) ||
			(lanceEquipeA == entities.LanceTesoura && lanceEquipeB == entities.LancePapel) {
			vencedor = entities.EquipePato
			resultadoSlug = "VITORIA_A"
		} else {
			vencedor = entities.EquipeGanso
			resultadoSlug = "VITORIA_B"
		}
	}

	res := &entities.PptResultResponse{
		Status:    entities.StatusResolvido,
		Rodada:    req.Rodada,
		Vencedor:  vencedor,
		Resultado: resultadoSlug,
		Lances: map[string]string{
			"A": lanceEquipeA,
			"B": lanceEquipeB,
		},
	}

	mem.Resultado = res
	return res, true, nil
}

// GetStatus retorna o estado presente da disputa de PPT
func (a *PptArbiterComponent) GetStatus(salaCodigo string, rodada int) *entities.PptStatusResponse {
	key := a.makeKey(salaCodigo, rodada)

	a.mu.RLock()
	defer a.mu.RUnlock()

	mem, exists := a.rodadas[key]
	if !exists {
		return &entities.PptStatusResponse{
			Status:          entities.StatusAguardandoLances,
			Rodada:          rodada,
			EquipesEnviadas: []string{},
		}
	}

	if mem.Resultado != nil {
		return &entities.PptStatusResponse{
			Status: mem.Resultado.Status,
			Rodada: rodada,
		}
	}

	equipes := make([]string, 0, len(mem.Lances))
	for _, l := range mem.Lances {
		equipes = append(equipes, l.Equipe)
	}

	return &entities.PptStatusResponse{
		Status:            entities.StatusAguardandoOponente,
		Rodada:            rodada,
		JogadoresEnviados: len(mem.Lances),
		EquipesEnviadas:   equipes,
	}
}
