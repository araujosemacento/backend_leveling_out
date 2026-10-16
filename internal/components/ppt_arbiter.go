package components

import (
	cryptorand "crypto/rand"
	"errors"
	"math/big"
	"sync"

	"leveling-out-backend/internal/entities"
)

// Erros de domínio da arbitragem
var (
	ErrDisputaInexistente = errors.New("disputa de PPT inexistente")
	ErrNaoRepresentante   = errors.New("apenas o representante da equipe pode realizar o lance")
	ErrLanceInvalido      = errors.New("lance inválido")
	ErrLanceDefinitivo    = errors.New("o lance da equipe já foi realizado e é definitivo")
)

// disputaPpt guarda em memória os lances sigilosos da disputa vigente de uma sala
type disputaPpt struct {
	representantes map[string]string // equipe -> player_id
	lances         map[string]string // equipe -> lance
}

// PptArbiterComponent modela o ArbitroPptComponent: guarda os lances em memória
// (sigilo até a resolução) e arbitra o confronto. É thread-safe.
type PptArbiterComponent struct {
	mu        sync.Mutex
	disputas  map[string]*disputaPpt // chave: código da sala
	sorteador func(n int) int
}

// NewPptArbiterComponent inicializa o árbitro de PPT em memória
func NewPptArbiterComponent() *PptArbiterComponent {
	return &PptArbiterComponent{
		disputas:  make(map[string]*disputaPpt),
		sorteador: sorteioSeguro,
	}
}

func sorteioSeguro(n int) int {
	v, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0
	}
	return int(v.Int64())
}

// Abrir inicia (ou reinicia) a disputa de uma sala, descartando lances anteriores
func (a *PptArbiterComponent) Abrir(sala, representanteA, representanteB string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.disputas[sala] = &disputaPpt{
		representantes: map[string]string{
			entities.EquipePato:  representanteA,
			entities.EquipeGanso: representanteB,
		},
		lances: make(map[string]string),
	}
}

// Garantir cria a disputa em memória caso ela não exista (ex.: após reinício do servidor)
func (a *PptArbiterComponent) Garantir(sala, representanteA, representanteB string) {
	a.mu.Lock()
	_, ok := a.disputas[sala]
	a.mu.Unlock()
	if !ok {
		a.Abrir(sala, representanteA, representanteB)
	}
}

// Fechar descarta a disputa em memória de uma sala
func (a *PptArbiterComponent) Fechar(sala string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.disputas, sala)
}

// SubmitMove registra o lance definitivo do representante da equipe.
// Retorna true quando ambos os lances já foram computados.
func (a *PptArbiterComponent) SubmitMove(sala, playerID, equipe, lance string) (bool, error) {
	if !entities.LanceValido(lance) {
		return false, ErrLanceInvalido
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	d, ok := a.disputas[sala]
	if !ok {
		return false, ErrDisputaInexistente
	}
	if d.representantes[equipe] == "" || d.representantes[equipe] != playerID {
		return false, ErrNaoRepresentante
	}
	if _, jaJogou := d.lances[equipe]; jaJogou {
		return false, ErrLanceDefinitivo
	}

	d.lances[equipe] = lance
	return len(d.lances) == 2, nil
}

// Preencher sorteia um lance aleatório para cada equipe que ainda não jogou (timeout ou desconexão).
// Retorna true quando, após o preenchimento, ambos os lances estão computados.
func (a *PptArbiterComponent) Preencher(sala string, equipes ...string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	d, ok := a.disputas[sala]
	if !ok {
		return false
	}
	opcoes := []string{entities.LancePedra, entities.LancePapel, entities.LanceTesoura}
	for _, eq := range equipes {
		if _, jaJogou := d.lances[eq]; !jaJogou {
			d.lances[eq] = opcoes[a.sorteador(len(opcoes))]
		}
	}
	return len(d.lances) == 2
}

// TemLance indica se a equipe já possui lance computado
func (a *PptArbiterComponent) TemLance(sala, equipe string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	d, ok := a.disputas[sala]
	if !ok {
		return false
	}
	_, jaJogou := d.lances[equipe]
	return jaJogou
}

// LanceDaEquipe devolve o lance sigiloso da equipe (uso exclusivo de rota autenticada da própria equipe)
func (a *PptArbiterComponent) LanceDaEquipe(sala, equipe string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	d, ok := a.disputas[sala]
	if !ok {
		return ""
	}
	return d.lances[equipe]
}

// Arbitrar resolve o confronto com os lances computados de ambas as equipes
func (a *PptArbiterComponent) Arbitrar(sala string) (*entities.PptResultResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	d, ok := a.disputas[sala]
	if !ok || len(d.lances) < 2 {
		return nil, ErrDisputaInexistente
	}
	lanceA := d.lances[entities.EquipePato]
	lanceB := d.lances[entities.EquipeGanso]
	resultado, vencedor := DecidirPpt(lanceA, lanceB)
	return &entities.PptResultResponse{
		Resultado: resultado,
		Vencedor:  vencedor,
		Lances:    map[string]string{entities.EquipePato: lanceA, entities.EquipeGanso: lanceB},
	}, nil
}

// DecidirPpt aplica as regras clássicas de Pedra, Papel e Tesoura.
// Retorna o slug do resultado e a equipe vencedora (vazia em caso de empate).
func DecidirPpt(lanceA, lanceB string) (string, string) {
	if lanceA == lanceB {
		return entities.PptResultadoEmpate, ""
	}
	venceA := (lanceA == entities.LancePedra && lanceB == entities.LanceTesoura) ||
		(lanceA == entities.LancePapel && lanceB == entities.LancePedra) ||
		(lanceA == entities.LanceTesoura && lanceB == entities.LancePapel)
	if venceA {
		return entities.PptResultadoVitoriaA, entities.EquipePato
	}
	return entities.PptResultadoVitoriaB, entities.EquipeGanso
}
