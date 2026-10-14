package hooks

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"leveling-out-backend/internal/entities"
)

// registerLobbyRoutes registra as rotas HTTP de handshake, presença e gestão de participantes no Lobby
func registerLobbyRoutes(se *core.ServeEvent, app core.App) {
	// Handshake Autoritativo e Ingresso Seguro (Zero Trust)
	se.Router.POST("/api/leveling/salas/{codigo}/entrar", func(e *core.RequestEvent) error {
		codigo := e.Request.PathValue("codigo")
		if codigo == "" {
			return e.BadRequestError("Código da sala é obrigatório.", nil)
		}

		type EntrarReq struct {
			Nome         string `json:"nome"`
			PlayerID     string `json:"playerId"`
			SessionToken string `json:"sessionToken"`
		}
		var req EntrarReq
		if err := e.BindBody(&req); err != nil {
			return e.BadRequestError("Corpo da requisição inválido.", err)
		}

		nomeLimpo := sanitizeNome(req.Nome)
		if len(nomeLimpo) < 2 {
			return e.BadRequestError("Apelido deve conter pelo menos 2 caracteres.", nil)
		}

		nowStr := time.Now().UTC().Format("2006-01-02 15:04:05.000Z")

		// Busca sala existente
		salas, _ := app.FindAllRecords("salas", dbx.HashExp{"codigo": codigo})

		var sala *core.Record
		isNovoHost := false

		if len(salas) == 0 {
			colecaoSalas, err := app.FindCollectionByNameOrId("salas")
			if err != nil {
				return e.InternalServerError("Coleção 'salas' não encontrada.", err)
			}
			sala = core.NewRecord(colecaoSalas)
			sala.Set("codigo", codigo)
			sala.Set("fase", entities.FaseLobby)
			sala.Set("rodada", 1)
			sala.Set("equipe_ativa", entities.EquipeCorvo)
			sala.Set("placar_a", 0)
			sala.Set("placar_b", 0)

			novoPlayerId := req.PlayerID
			if novoPlayerId == "" {
				novoPlayerId = "p_" + generateSecureToken()[:10]
			}
			sala.Set("host_id", novoPlayerId)
			if err := app.Save(sala); err != nil {
				return e.InternalServerError("Falha ao criar sala no banco.", err)
			}
			isNovoHost = true
			req.PlayerID = novoPlayerId
		} else {
			sala = salas[0]
		}

		// Busca participantes da sala
		jogadores, _ := app.FindAllRecords("jogadores", dbx.HashExp{"sala_codigo": codigo})

		// Subcaso A: Jogador já possui playerId conhecido
		if req.PlayerID != "" {
			for _, jg := range jogadores {
				if jg.GetString("player_id") == req.PlayerID {
					tokenArmazenado := jg.GetString("session_token")
					if tokenArmazenado != "" && req.SessionToken != "" && tokenArmazenado != req.SessionToken {
						return e.UnauthorizedError("Credenciais de sessão inválidas.", nil)
					}

					novoToken := tokenArmazenado
					if novoToken == "" {
						novoToken = generateSecureToken()
						jg.Set("session_token", novoToken)
					}
					jg.Set("nome", nomeLimpo)
					jg.Set("last_seen", nowStr)
					_ = app.Save(jg)

					snapBytes, _ := fetchRoomSnapshotRaw(app, codigo)
					var snapMsg any
					if len(snapBytes) > 0 {
						snapMsg = json.RawMessage(snapBytes)
					}

					return e.JSON(200, map[string]any{
						"success":      true,
						"salaCodigo":   codigo,
						"playerId":     jg.GetString("player_id"),
						"sessionToken": novoToken,
						"isHost":       jg.GetBool("is_host"),
						"equipe":       jg.GetString("equipe"),
						"reconectado":  true,
						"snapshot":     snapMsg,
					})
				}
			}
		}

		// Subcaso B: Reconexão de outro dispositivo pelo mesmo apelido
		for _, jg := range jogadores {
			if strings.EqualFold(jg.GetString("nome"), nomeLimpo) {
				lastSeenTime, _ := time.Parse("2006-01-02 15:04:05.000Z", jg.GetString("last_seen"))
				if lastSeenTime.IsZero() {
					lastSeenTime, _ = time.Parse("2006-01-02 15:04:05", jg.GetString("last_seen"))
				}

				// Se tiver ping recente (< 10s), está ativo no aparelho antigo: bloqueia
				if !lastSeenTime.IsZero() && time.Since(lastSeenTime) <= 10*time.Second {
					return e.BadRequestError("Este apelido já está ativo na partida.", nil)
				}

				novoPlayerId := req.PlayerID
				if novoPlayerId == "" {
					novoPlayerId = "p_" + generateSecureToken()[:10]
				}
				novoToken := generateSecureToken()

				isHost := jg.GetBool("is_host")
				jg.Set("player_id", novoPlayerId)
				jg.Set("session_token", novoToken)
				jg.Set("last_seen", nowStr)
				_ = app.Save(jg)

				if isHost {
					sala.Set("host_id", novoPlayerId)
					_ = app.Save(sala)
				}

				log.Printf("[HANDSHAKE] Assento de '%s' reassumido na sala '%s'", nomeLimpo, codigo)

				snapBytes, _ := fetchRoomSnapshotRaw(app, codigo)
				var snapMsg any
				if len(snapBytes) > 0 {
					snapMsg = json.RawMessage(snapBytes)
				}

				return e.JSON(200, map[string]any{
					"success":      true,
					"salaCodigo":   codigo,
					"playerId":     novoPlayerId,
					"sessionToken": novoToken,
					"isHost":       isHost,
					"equipe":       jg.GetString("equipe"),
					"reassumido":   true,
					"snapshot":     snapMsg,
				})
			}
		}

		// Subcaso C: Novo jogador
		if len(jogadores) >= 4 {
			return e.BadRequestError("A sala já atingiu a capacidade máxima de 4 jogadores.", nil)
		}

		novoPlayerId := req.PlayerID
		if novoPlayerId == "" {
			novoPlayerId = "p_" + generateSecureToken()[:10]
		}
		novoToken := generateSecureToken()

		colecaoJogadores, err := app.FindCollectionByNameOrId("jogadores")
		if err != nil {
			return e.InternalServerError("Coleção 'jogadores' não encontrada.", err)
		}

		isHost := isNovoHost || len(jogadores) == 0
		novoJogador := core.NewRecord(colecaoJogadores)
		novoJogador.Set("sala_codigo", codigo)
		novoJogador.Set("player_id", novoPlayerId)
		novoJogador.Set("nome", nomeLimpo)
		novoJogador.Set("equipe", entities.EquipeAleatorio)
		novoJogador.Set("is_host", isHost)
		novoJogador.Set("session_token", novoToken)
		novoJogador.Set("papel", "ESPECTADOR")
		novoJogador.Set("last_seen", nowStr)

		if err := app.Save(novoJogador); err != nil {
			return e.InternalServerError("Falha ao registrar jogador.", err)
		}

		if isHost && sala.GetString("host_id") != novoPlayerId {
			sala.Set("host_id", novoPlayerId)
			_ = app.Save(sala)
		}

		log.Printf("[HANDSHAKE] Jogador '%s' (%s) ingressou na sala '%s' (Host: %v)", nomeLimpo, novoPlayerId, codigo, isHost)

		snapBytes, _ := fetchRoomSnapshotRaw(app, codigo)
		var snapMsg any
		if len(snapBytes) > 0 {
			snapMsg = json.RawMessage(snapBytes)
		}

		return e.JSON(201, map[string]any{
			"success":      true,
			"salaCodigo":   codigo,
			"playerId":     novoPlayerId,
			"sessionToken": novoToken,
			"isHost":       isHost,
			"equipe":       entities.EquipeAleatorio,
			"reconectado":  false,
			"snapshot":     snapMsg,
		})
	})

	// Ping de Heartbeat e Presença em Tempo Real
	se.Router.POST("/api/leveling/salas/{codigo}/ping", func(e *core.RequestEvent) error {
		codigo := e.Request.PathValue("codigo")
		playerId := e.Request.Header.Get("X-Player-Id")
		sessionToken := e.Request.Header.Get("X-Session-Token")

		if codigo == "" || playerId == "" {
			return e.BadRequestError("Identificação de sala e jogador obrigatória.", nil)
		}

		jogadores, err := app.FindAllRecords("jogadores", dbx.HashExp{
			"sala_codigo": codigo,
			"player_id":   playerId,
		})
		if err != nil || len(jogadores) == 0 {
			return e.NotFoundError("Jogador não encontrado.", nil)
		}

		jogador := jogadores[0]
		tokenArmazenado := jogador.GetString("session_token")
		if tokenArmazenado != "" && sessionToken != "" && tokenArmazenado != sessionToken {
			return e.UnauthorizedError("Token de sessão inválido ou expirado.", nil)
		}

		nowStr := time.Now().UTC().Format("2006-01-02 15:04:05.000Z")
		_, _ = app.DB().Update("jogadores", dbx.Params{
			"last_seen": nowStr,
		}, dbx.HashExp{"id": jogador.Id}).Execute()

		return e.JSON(200, map[string]any{
			"success":   true,
			"timestamp": nowStr,
		})
	})

	// Troca de Equipe com Validação Autoritativa
	se.Router.POST("/api/leveling/salas/{codigo}/equipe", func(e *core.RequestEvent) error {
		codigo := e.Request.PathValue("codigo")
		playerId := e.Request.Header.Get("X-Player-Id")
		sessionToken := e.Request.Header.Get("X-Session-Token")

		type EquipeReq struct {
			Equipe string `json:"equipe"`
		}
		var req EquipeReq
		if err := e.BindBody(&req); err != nil {
			return e.BadRequestError("Corpo da requisição inválido.", err)
		}

		jogadores, err := app.FindAllRecords("jogadores", dbx.HashExp{
			"sala_codigo": codigo,
			"player_id":   playerId,
		})
		if err != nil || len(jogadores) == 0 {
			return e.NotFoundError("Jogador não encontrado.", nil)
		}

		jogador := jogadores[0]
		if tok := jogador.GetString("session_token"); tok != "" && sessionToken != "" && tok != sessionToken {
			return e.UnauthorizedError("Token de sessão inválido.", nil)
		}

		novaEquipe := req.Equipe
		if novaEquipe != entities.EquipeCorvo && novaEquipe != entities.EquipeAguia {
			novaEquipe = entities.EquipeAleatorio
		}

		if novaEquipe == entities.EquipeCorvo || novaEquipe == entities.EquipeAguia {
			outros, _ := app.FindAllRecords("jogadores",
				dbx.HashExp{"sala_codigo": codigo, "equipe": novaEquipe},
				dbx.NewExp("player_id != {:pid}", dbx.Params{"pid": playerId}),
			)
			if len(outros) >= 2 {
				return e.BadRequestError("Esta equipe já atingiu o limite de 2 integrantes.", nil)
			}
		}

		jogador.Set("equipe", novaEquipe)
		jogador.Set("last_seen", time.Now().UTC().Format("2006-01-02 15:04:05.000Z"))
		if err := app.Save(jogador); err != nil {
			return e.InternalServerError("Falha ao atualizar equipe.", err)
		}

		return e.JSON(200, map[string]any{
			"success": true,
			"equipe":  novaEquipe,
		})
	})

	// Desfazer Sala (Exclusivo do Anfitrião)
	se.Router.POST("/api/leveling/salas/{codigo}/desfazer", func(e *core.RequestEvent) error {
		codigo := e.Request.PathValue("codigo")
		playerId := e.Request.Header.Get("X-Player-Id")

		salas, err := app.FindAllRecords("salas", dbx.HashExp{"codigo": codigo})
		if err != nil || len(salas) == 0 {
			return e.NotFoundError("Sala não encontrada.", nil)
		}

		sala := salas[0]
		if sala.GetString("host_id") != playerId {
			return e.ForbiddenError("Apenas o anfitrião pode desfazer a sala.", nil)
		}

		sala.Set("fase", entities.FaseFimJogo)
		sala.Set("motivo_fim", "DISSOLVIDA_HOST")
		_ = app.Save(sala)

		log.Printf("[SALA] Sala '%s' desfeita pelo anfitrião '%s'", codigo, playerId)
		return e.JSON(200, map[string]any{"success": true})
	})
}
