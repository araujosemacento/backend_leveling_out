package actor

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"sync/atomic"
	"time"

	"leveling-out-backend/internal/components"
	"leveling-out-backend/internal/entities"
	"leveling-out-backend/internal/protocol"
)

// EventoInterno encapsula uma ação enfileirada para o loop da sala
type EventoInterno struct {
	Tipo     string
	PlayerID string
	Conn     *ClientConn
	Payload  map[string]interface{}
	RespChan chan error
}

// RoomActor representa a entidade soberana e autônoma de uma sala em memória
type RoomActor struct {
	codigo   string
	estado   entities.SalaEstado
	conexoes map[string]*ClientConn // player_id -> conexão ativa
	inbox    chan EventoInterno
	fechar   chan struct{}

	// Componentes Atômicos acoplados
	pptArbiter   *components.PptArbiterComponent
	proximity    *components.ProximityComponent
	secretTarget *components.SecretTargetComponent
	spectrums    *components.SpectrumsComponent

	// Timers internos de transição automática
	timerTransicao time.Time
	acaoTransicao  string

	// Opções de cartas ativas para a fase de escolha
	cartasOpcoes []components.SpectrumCard

	// Rastreamento de alternância compulsória por equipe (equipe -> playerID)
	ultimoCodificador map[string]string
	ultimoLider       map[string]string

	// Listas autoritativas de banimento da sala atual
	dispositivosBanidos map[string]bool
	jogadoresBanidos     map[string]bool

	// Controle de inatividade e métricas thread-safe
	ultimaAtividade time.Time
	ultimoAcesso    atomic.Int64 // Unix timestamp em segundos
	conexoesAtivas  atomic.Int32
	totalJogadores  atomic.Int32
	fecharOnce      sync.Once
}

// NewRoomActor instancia uma nova sala com seus componentes atômicos
func NewRoomActor(codigo string) *RoomActor {
	actor := &RoomActor{
		codigo: codigo,
		estado: entities.SalaEstado{
			Codigo:      codigo,
			Fase:        entities.FaseLobby,
			RodadaAtual: 1,
			PlacarA:     0,
			PlacarB:     0,
			Jogadores:   make(map[string]*entities.Jogador),
		},
		conexoes:            make(map[string]*ClientConn),
		inbox:               make(chan EventoInterno, 256),
		fechar:              make(chan struct{}),
		pptArbiter:          components.NewPptArbiterComponent(),
		proximity:           components.NewProximityComponent(),
		secretTarget:        components.NewSecretTargetComponent(),
		spectrums:           components.NewSpectrumsComponent(),
		ultimoCodificador:   make(map[string]string),
		ultimoLider:         make(map[string]string),
		dispositivosBanidos: make(map[string]bool),
		jogadoresBanidos:     make(map[string]bool),
		ultimaAtividade:     time.Now(),
	}
	actor.ultimoAcesso.Store(time.Now().Unix())
	actor.conexoesAtivas.Store(0)

	go actor.loop()
	return actor
}

// Despachar enfileira um evento de forma assíncrona
func (r *RoomActor) Despachar(tipo, playerID string, payload map[string]interface{}, conn *ClientConn) {
	r.inbox <- EventoInterno{
		Tipo:     tipo,
		PlayerID: playerID,
		Payload:  payload,
		Conn:     conn,
	}
}

// DespacharSincrono enfileira um evento e aguarda a conclusão do processamento no loop
func (r *RoomActor) DespacharSincrono(tipo, playerID string, payload map[string]interface{}, conn *ClientConn) error {
	resp := make(chan error, 1)
	r.inbox <- EventoInterno{
		Tipo:     tipo,
		PlayerID: playerID,
		Payload:  payload,
		Conn:     conn,
		RespChan: resp,
	}
	return <-resp
}

// loop processa os eventos sequencialmente na goroutine dona da sala (Zero Race Conditions)
func (r *RoomActor) loop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case ev := <-r.inbox:
			r.processarEvento(ev)
			if ev.RespChan != nil {
				ev.RespChan <- nil
			}

		case <-ticker.C:
			r.processarTicksTemporais()

		case <-r.fechar:
			r.encerrar()
			return
		}
	}
}

func (r *RoomActor) processarEvento(ev EventoInterno) {
	agora := time.Now()
	r.ultimaAtividade = agora
	r.ultimoAcesso.Store(agora.Unix())

	switch ev.Tipo {
	case "CONECTAR":
		r.handleConectar(ev.Conn)
	case "DESCONECTAR":
		r.handleDesconectar(ev.PlayerID)
	case protocol.TipoEscolherEquipe:
		r.handleEscolherEquipe(ev.PlayerID, ev.Payload)
	case protocol.TipoIniciarPartida:
		r.handleIniciarPartida(ev.PlayerID)
	case protocol.TipoExpulsarJogador:
		r.handleExpulsarJogador(ev.PlayerID, ev.Payload)
	case protocol.TipoBanirJogador:
		r.handleBanirJogador(ev.PlayerID, ev.Payload)
	case protocol.TipoDesfazerPartida:
		r.handleDesfazerPartida(ev.PlayerID)
	case protocol.TipoLancePpt:
		r.handleLancePpt(ev.PlayerID, ev.Payload)
	case protocol.TipoPreSelecionarEspectro:
		r.handlePreSelecionarEspectro(ev.PlayerID, ev.Payload)
	case protocol.TipoVotarEspectro:
		r.handleVotarEspectro(ev.PlayerID, ev.Payload)
	case protocol.TipoEscolherEspectro:
		r.handleEscolherEspectro(ev.PlayerID, ev.Payload)
	case protocol.TipoSubmeterDica:
		r.handleSubmeterDica(ev.PlayerID, ev.Payload)
	case protocol.TipoSliderMover:
		r.handleSliderMover(ev.PlayerID, ev.Payload)
	case protocol.TipoConfirmarPalpite:
		r.handleConfirmarPalpite(ev.PlayerID)
	case protocol.TipoRevanche:
		r.handleRevanche(ev.PlayerID)
	}
}

func (r *RoomActor) encerrar() {
	for _, conn := range r.conexoes {
		conn.Fechar()
	}
	r.conexoes = make(map[string]*ClientConn)
	r.conexoesAtivas.Store(0)
}

// ConexoesAtivas retorna a quantidade de sockets WebSocket conectados atualmente (thread-safe).
func (r *RoomActor) ConexoesAtivas() int {
	return int(r.conexoesAtivas.Load())
}

// TotalJogadores retorna o total de jogadores cadastrados no estado da sala (thread-safe).
func (r *RoomActor) TotalJogadores() int {
	return int(r.totalJogadores.Load())
}

// EstaVazia verifica se a sala está sem conexões e sem jogadores cadastrados (sessão abandonada).
func (r *RoomActor) EstaVazia() bool {
	return r.conexoesAtivas.Load() == 0 && r.totalJogadores.Load() == 0
}

// UltimoAcesso retorna o horário do último evento ou criação do ator (thread-safe).
func (r *RoomActor) UltimoAcesso() time.Time {
	return time.Unix(r.ultimoAcesso.Load(), 0)
}

// EstaInativo avalia de forma thread-safe se a sala está sem conexões ativas
// e se o tempo desde a última atividade ultrapassou o limite especificado.
func (r *RoomActor) EstaInativo(limite time.Duration) bool {
	if r.conexoesAtivas.Load() > 0 {
		return false
	}
	ultimo := time.Unix(r.ultimoAcesso.Load(), 0)
	return time.Since(ultimo) >= limite
}

// Encerrar desliga o loop autônomo do ator e libera conexões de forma idempotente e thread-safe.
func (r *RoomActor) Encerrar() {
	r.fecharOnce.Do(func() {
		close(r.fechar)
	})
}

func gerarTokenSeguro() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func gerarIDCurto() string {
	bytes := make([]byte, 4)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
