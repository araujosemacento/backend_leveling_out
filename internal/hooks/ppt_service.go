package hooks

import (
	"context"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"leveling-out-backend/internal/components"
	"leveling-out-backend/internal/entities"
)

// Parâmetros temporais autoritativos da disputa de iniciativa
const (
	pptPrazoEscolha      = 10 * time.Second
	pptDuracaoContagem   = 3 * time.Second
	pptExibicaoVitoria   = 4 * time.Second
	pptExibicaoEmpate    = 2500 * time.Millisecond
	pptLimiteInatividade = 15 * time.Second
	pptIntervaloTick     = 250 * time.Millisecond
	formatoDataPB        = "2006-01-02 15:04:05.000Z"
)

// pptHTTPError associa um status HTTP a uma falha de regra de negócio da disputa
type pptHTTPError struct {
	Status  int
	Message string
}

// pptService orquestra o ciclo da disputa de PPT: prazo, contagem, revelação e avanço de fase.
// O estado público vive nos registros da sala (SSE); os lances sigilosos vivem apenas no árbitro.
type pptService struct {
	app     core.App
	arbiter *components.PptArbiterComponent
	mu      sync.Mutex
}

func newPptService(app core.App, arbiter *components.PptArbiterComponent) *pptService {
	return &pptService{app: app, arbiter: arbiter}
}

func formatarData(t time.Time) string {
	return t.UTC().Format(formatoDataPB)
}

// escolherRepresentante aplica a regra de iniciativa: o primeiro a ingressar na equipe
// (jogador manual com menor equipe_desde) representa; na ausência, o primeiro sorteado.
func escolherRepresentante(manuais []*core.Record, sorteados []*core.Record) string {
	if len(manuais) > 0 {
		sort.SliceStable(manuais, func(i, j int) bool {
			ti := manuais[i].GetDateTime("equipe_desde").Time()
			tj := manuais[j].GetDateTime("equipe_desde").Time()
			if ti.IsZero() != tj.IsZero() {
				return !ti.IsZero()
			}
			return ti.Before(tj)
		})
		return manuais[0].GetString("player_id")
	}
	if len(sorteados) > 0 {
		return sorteados[0].GetString("player_id")
	}
	return ""
}

// abrirEscolha prepara os campos públicos da sala para uma nova janela de escolha (sem salvar)
func (s *pptService) abrirEscolha(sala *core.Record) {
	sala.Set("ppt_estado", entities.PptEstadoEscolha)
	sala.Set("ppt_prazo", formatarData(time.Now().Add(pptPrazoEscolha)))
	sala.Set("ppt_revelacao_em", "")
	sala.Set("ppt_enviou_a", false)
	sala.Set("ppt_enviou_b", false)
	sala.Set("ppt_resultado", "")
	sala.Set("ppt_lance_a", "")
	sala.Set("ppt_lance_b", "")
}

// Iniciar registra os representantes e abre a primeira escolha (sem salvar a sala)
func (s *pptService) Iniciar(sala *core.Record, repA, repB string) {
	sala.Set("ppt_representante_a", repA)
	sala.Set("ppt_representante_b", repB)
	s.arbiter.Abrir(sala.GetString("codigo"), repA, repB)
	s.abrirEscolha(sala)
}

func (s *pptService) iniciarContagem(sala *core.Record) {
	sala.Set("ppt_estado", entities.PptEstadoContagem)
	sala.Set("ppt_revelacao_em", formatarData(time.Now().Add(pptDuracaoContagem)))
}

// Enviar registra o lance definitivo do representante autenticado
func (s *pptService) Enviar(codigo string, jogador *core.Record, lance string) *pptHTTPError {
	s.mu.Lock()
	defer s.mu.Unlock()

	sala, err := s.app.FindFirstRecordByData("salas", "codigo", codigo)
	if err != nil {
		return &pptHTTPError{404, "Sala não encontrada."}
	}
	if sala.GetString("fase") != entities.FaseIniciativa || sala.GetString("ppt_estado") != entities.PptEstadoEscolha {
		return &pptHTTPError{409, "A disputa não está aberta para lances."}
	}
	if prazo := sala.GetDateTime("ppt_prazo").Time(); !prazo.IsZero() && time.Now().After(prazo) {
		return &pptHTTPError{409, "O prazo para o lance foi encerrado."}
	}

	equipe := jogador.GetString("equipe")
	if equipe != entities.EquipePato && equipe != entities.EquipeGanso {
		return &pptHTTPError{403, "Apenas representantes de equipe podem realizar o lance."}
	}

	s.arbiter.Garantir(codigo, sala.GetString("ppt_representante_a"), sala.GetString("ppt_representante_b"))
	pronto, err := s.arbiter.SubmitMove(codigo, jogador.GetString("player_id"), equipe, lance)
	switch err {
	case nil:
	case components.ErrNaoRepresentante:
		return &pptHTTPError{403, err.Error()}
	case components.ErrLanceDefinitivo:
		return &pptHTTPError{409, err.Error()}
	case components.ErrLanceInvalido:
		return &pptHTTPError{400, err.Error()}
	default:
		return &pptHTTPError{500, err.Error()}
	}

	sala.Set("ppt_enviou_"+sufixoEquipe(equipe), true)
	if pronto {
		s.iniciarContagem(sala)
	}
	if err := s.app.Save(sala); err != nil {
		return &pptHTTPError{500, "Falha ao registrar o lance."}
	}
	return nil
}

// LanceDaEquipe devolve o lance sigiloso da equipe do jogador autenticado
func (s *pptService) LanceDaEquipe(codigo string, jogador *core.Record) string {
	return s.arbiter.LanceDaEquipe(codigo, jogador.GetString("equipe"))
}

func sufixoEquipe(equipe string) string {
	if equipe == entities.EquipePato {
		return "a"
	}
	return "b"
}

// Executar roda o laço de avanço temporal até o cancelamento do contexto
func (s *pptService) Executar(ctx context.Context) {
	ticker := time.NewTicker(pptIntervaloTick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

func (s *pptService) tick() {
	salas, err := s.app.FindAllRecords("salas", dbx.HashExp{"fase": entities.FaseIniciativa})
	if err != nil {
		return
	}
	for _, sala := range salas {
		s.processar(sala.GetString("codigo"))
	}
}

func (s *pptService) processar(codigo string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sala, err := s.app.FindFirstRecordByData("salas", "codigo", codigo)
	if err != nil || sala.GetString("fase") != entities.FaseIniciativa {
		return
	}
	now := time.Now()

	switch sala.GetString("ppt_estado") {
	case entities.PptEstadoEscolha:
		s.processarEscolha(sala, now)
	case entities.PptEstadoContagem:
		if rev := sala.GetDateTime("ppt_revelacao_em").Time(); !rev.IsZero() && !now.Before(rev) {
			s.revelar(sala)
		}
	case entities.PptEstadoResultado:
		s.processarResultado(sala, now)
	}
}

func (s *pptService) processarEscolha(sala *core.Record, now time.Time) {
	codigo := sala.GetString("codigo")
	repA := sala.GetString("ppt_representante_a")
	repB := sala.GetString("ppt_representante_b")
	s.arbiter.Garantir(codigo, repA, repB)

	var faltantes []string
	if prazo := sala.GetDateTime("ppt_prazo").Time(); !prazo.IsZero() && !now.Before(prazo) {
		// Timeout: o servidor sorteia o lance de quem não jogou
		faltantes = []string{entities.EquipePato, entities.EquipeGanso}
	} else {
		// Desconexão do representante tem o mesmo tratamento do timeout
		for equipe, rep := range map[string]string{entities.EquipePato: repA, entities.EquipeGanso: repB} {
			if !s.arbiter.TemLance(codigo, equipe) && s.representanteInativo(codigo, rep, now) {
				log.Printf("[PPT] Representante '%s' da sala '%s' inativo: lance aleatório.", rep, codigo)
				faltantes = append(faltantes, equipe)
			}
		}
	}

	pronto := false
	if len(faltantes) > 0 {
		pronto = s.arbiter.Preencher(codigo, faltantes...)
	}

	alterou := false
	for _, eq := range []string{entities.EquipePato, entities.EquipeGanso} {
		campo := "ppt_enviou_" + sufixoEquipe(eq)
		tem := s.arbiter.TemLance(codigo, eq)
		if sala.GetBool(campo) != tem {
			sala.Set(campo, tem)
			alterou = true
		}
	}
	if pronto {
		s.iniciarContagem(sala)
		alterou = true
	}
	if alterou {
		_ = s.app.Save(sala)
	}
}

func (s *pptService) representanteInativo(codigo, playerID string, now time.Time) bool {
	jogadores, err := s.app.FindAllRecords("jogadores", dbx.HashExp{"sala_codigo": codigo, "player_id": playerID})
	if err != nil || len(jogadores) == 0 {
		// Representante removido da sala: tratado como inativo
		return true
	}
	visto := jogadores[0].GetDateTime("last_seen").Time()
	return !visto.IsZero() && now.Sub(visto) > pptLimiteInatividade
}

func (s *pptService) revelar(sala *core.Record) {
	res, err := s.arbiter.Arbitrar(sala.GetString("codigo"))
	if err != nil {
		// Estado em memória perdido (ex.: reinício do servidor): reabre a escolha
		s.arbiter.Abrir(sala.GetString("codigo"), sala.GetString("ppt_representante_a"), sala.GetString("ppt_representante_b"))
		s.abrirEscolha(sala)
		_ = s.app.Save(sala)
		return
	}
	sala.Set("ppt_estado", entities.PptEstadoResultado)
	sala.Set("ppt_resultado", res.Resultado)
	sala.Set("ppt_lance_a", res.Lances[entities.EquipePato])
	sala.Set("ppt_lance_b", res.Lances[entities.EquipeGanso])
	_ = s.app.Save(sala)
}

func (s *pptService) processarResultado(sala *core.Record, now time.Time) {
	rev := sala.GetDateTime("ppt_revelacao_em").Time()
	resultado := sala.GetString("ppt_resultado")
	espera := pptExibicaoVitoria
	if resultado == entities.PptResultadoEmpate {
		espera = pptExibicaoEmpate
	}
	if rev.IsZero() || now.Before(rev.Add(espera)) {
		return
	}

	codigo := sala.GetString("codigo")
	switch resultado {
	case entities.PptResultadoEmpate:
		s.arbiter.Abrir(codigo, sala.GetString("ppt_representante_a"), sala.GetString("ppt_representante_b"))
		s.abrirEscolha(sala)
	default:
		vencedor := entities.EquipePato
		if resultado == entities.PptResultadoVitoriaB {
			vencedor = entities.EquipeGanso
		}
		s.arbiter.Fechar(codigo)
		sala.Set("equipe_ativa", vencedor)
		sala.Set("fase", entities.FaseRodadaEspectro)
		sala.Set("ppt_estado", "")
		log.Printf("[PPT] Sala '%s': equipe %s inicia a primeira rodada.", codigo, vencedor)
	}
	_ = s.app.Save(sala)
}
