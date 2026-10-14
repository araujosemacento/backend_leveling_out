package hooks

import (
	cryptorand "crypto/rand"
	"log"
	"math/big"
	"strconv"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"leveling-out-backend/internal/components"
	"leveling-out-backend/internal/entities"
)

// RegisterGameHooks registra todos os componentes atômicos de regras, árbitros e cron jobs no PocketBase
func RegisterGameHooks(app core.App) {
	proximity := components.NewProximityComponent()
	secretTarget := components.NewSecretTargetComponent()
	pptArbiter := components.NewPptArbiterComponent()
	cleanup := components.NewCleanupComponent()

	// 1. Regras autoritativas para a coleção salas (Sorteio de meta, validação de dica e cálculo na revelação)
	app.OnRecordUpdateRequest("salas").BindFunc(func(e *core.RecordRequestEvent) error {
		fase := e.Record.GetString("fase")

		// Sorteia meta oculta se na fase de DICA ou ESCOLHA_ESPECTRO e meta ainda não definida
		if fase == entities.FaseRodadaDica || fase == entities.FaseRodadaEspectro || fase == "DICA" {
			metaAtual := e.Record.GetInt("meta_oculta")
			if metaAtual <= 0 {
				meta := secretTarget.Generate()
				e.Record.Set("meta_oculta", meta)
				e.Record.Set("pontos_rodada", 0)
				log.Printf("[HOOK SALAS] Nova meta_oculta sorteada para sala '%s': %d", e.Record.GetString("codigo"), meta)
			}
		}

		// Validação ao avançar para palpite: exige dica preenchida
		if fase == entities.FaseRodadaPalpite {
			dica := e.Record.GetString("dica")
			if dica == "" {
				return e.BadRequestError("Não é possível avançar para palpite sem uma dica cadastrada.", nil)
			}
		}

		// Fase de Revelação: Cálculo autoritativo de proximidade e atualização do placar
		if fase == entities.FaseRodadaRevelada {
			meta := e.Record.GetInt("meta_oculta")
			palpite := e.Record.GetInt("palpite")

			res := proximity.Calculate(meta, palpite)
			e.Record.Set("pontos_rodada", res.Pontos)

			equipe := e.Record.GetString("equipe_ativa")
			targetVitoria := entities.MetaPontuacaoPadrao

			switch equipe {
			case entities.EquipeCorvo:
				novoPlacar := e.Record.GetInt("placar_a") + res.Pontos
				e.Record.Set("placar_a", novoPlacar)
				if novoPlacar >= targetVitoria {
					e.Record.Set("vencedor", entities.EquipeCorvo)
					e.Record.Set("fase", entities.FaseFimJogo)
				}
			case entities.EquipeAguia:
				novoPlacar := e.Record.GetInt("placar_b") + res.Pontos
				e.Record.Set("placar_b", novoPlacar)
				if novoPlacar >= targetVitoria {
					e.Record.Set("vencedor", entities.EquipeAguia)
					e.Record.Set("fase", entities.FaseFimJogo)
				}
			}

			log.Printf("[GAME_RULES] Revelação da sala '%s': Meta=%d, Palpite=%d -> %d pts (%s)",
				e.Record.GetString("codigo"), meta, palpite, res.Pontos, res.Label)
		}

		return e.Next()
	})

	// 2. Criação de nova rodada: sorteia meta oculta se na fase de dica ou espectro
	app.OnRecordCreateRequest("rodadas").BindFunc(func(e *core.RecordRequestEvent) error {
		fase := e.Record.GetString("fase_rodada")

		if fase == entities.FaseRodadaDica || fase == entities.FaseRodadaEspectro {
			metaAtual := e.Record.GetInt("meta_oculta")
			if metaAtual <= 0 {
				meta := secretTarget.Generate()
				e.Record.Set("meta_oculta", meta)
				e.Record.Set("pontos", 0)
			}
		}

		return e.Next()
	})

	// 2. Atualização e progressão da rodada: validações, cálculo de pontos e atualização em cascata
	app.OnRecordUpdateRequest("rodadas").BindFunc(func(e *core.RecordRequestEvent) error {
		fase := e.Record.GetString("fase_rodada")

		// Sorteia meta oculta se ainda não existir
		if fase == entities.FaseRodadaDica || fase == entities.FaseRodadaEspectro {
			metaAtual := e.Record.GetInt("meta_oculta")
			if metaAtual <= 0 {
				meta := secretTarget.Generate()
				e.Record.Set("meta_oculta", meta)
				e.Record.Set("pontos", 0)
			}
		}

		// Validação ao avançar para palpite: exige dica preenchida
		if fase == entities.FaseRodadaPalpite {
			dica := e.Record.GetString("dica")
			if dica == "" {
				return e.BadRequestError("Não é possível avançar para palpite sem uma dica cadastrada.", nil)
			}
		}

		// Fase de Revelação: Cálculo autoritativo de proximidade e atualização do placar da sala
		if fase == entities.FaseRodadaRevelada {
			meta := e.Record.GetInt("meta_oculta")
			palpite := e.Record.GetInt("palpite")

			res := proximity.Calculate(meta, palpite)
			e.Record.Set("pontos", res.Pontos)

			salaCodigo := e.Record.GetString("sala_codigo")
			equipe := e.Record.GetString("equipe")

			salas, err := app.FindRecordsByFilter(
				"salas",
				"codigo = {:codigo}",
				"-created",
				1,
				0,
				dbx.Params{"codigo": salaCodigo},
			)
			if err == nil && len(salas) > 0 {
				sala := salas[0]
				targetVitoria := entities.MetaPontuacaoPadrao

				switch equipe {
				case entities.EquipeCorvo:
					novoPlacar := sala.GetInt("placar_a") + res.Pontos
					sala.Set("placar_a", novoPlacar)
					if novoPlacar >= targetVitoria {
						sala.Set("vencedor", entities.EquipeCorvo)
						sala.Set("fase", entities.FaseFimJogo)
					}
				case entities.EquipeAguia:
					novoPlacar := sala.GetInt("placar_b") + res.Pontos
					sala.Set("placar_b", novoPlacar)
					if novoPlacar >= targetVitoria {
						sala.Set("vencedor", entities.EquipeAguia)
						sala.Set("fase", entities.FaseFimJogo)
					}
				}

				if errSave := app.Save(sala); errSave != nil {
					log.Printf("[GAME_RULES ERROR] Falha ao atualizar placar da sala '%s': %v", salaCodigo, errSave)
				} else {
					log.Printf("[GAME_RULES] Placar da sala '%s' atualizado: Equipe %s +%d pts (%s)", salaCodigo, equipe, res.Pontos, res.Label)
				}
			}
		}

		return e.Next()
	})

	// 3. Regras autoritativas para a coleção jogadores (Quórum máximo de 4, controle de anfitrião e lotação de equipes)
	app.OnRecordCreateRequest("jogadores").BindFunc(func(e *core.RecordRequestEvent) error {
		salaCodigo := e.Record.GetString("sala_codigo")
		if salaCodigo == "" {
			return e.BadRequestError("sala_codigo é obrigatório.", nil)
		}

		jogadoresExistentes, _ := app.FindRecordsByFilter(
			"jogadores",
			"sala_codigo = {:codigo}",
			"-created",
			10,
			0,
			dbx.Params{"codigo": salaCodigo},
		)

		if len(jogadoresExistentes) >= 4 {
			return e.BadRequestError("A sala já atingiu a capacidade máxima de 4 jogadores.", nil)
		}

		// O primeiro jogador que entrar na sala torna-se automaticamente o dono (host)
		if len(jogadoresExistentes) == 0 {
			e.Record.Set("is_host", true)
			salas, _ := app.FindRecordsByFilter("salas", "codigo = {:codigo}", "-created", 1, 0, dbx.Params{"codigo": salaCodigo})
			if len(salas) > 0 {
				salas[0].Set("host_id", e.Record.GetString("player_id"))
				_ = app.Save(salas[0])
			}
		}

		// Equipe padrão é ALEATORIO se não informada ou se a equipe solicitada já estiver cheia
		eq := e.Record.GetString("equipe")
		if eq != entities.EquipeCorvo && eq != entities.EquipeAguia {
			e.Record.Set("equipe", entities.EquipeAleatorio)
		} else {
			count := 0
			for _, jg := range jogadoresExistentes {
				if jg.GetString("equipe") == eq {
					count++
				}
			}
			if count >= 2 {
				e.Record.Set("equipe", entities.EquipeAleatorio)
			}
		}

		return e.Next()
	})

	app.OnRecordUpdateRequest("jogadores").BindFunc(func(e *core.RecordRequestEvent) error {
		salaCodigo := e.Record.GetString("sala_codigo")
		salas, _ := app.FindRecordsByFilter("salas", "codigo = {:codigo}", "-created", 1, 0, dbx.Params{"codigo": salaCodigo})
		if len(salas) > 0 && salas[0].GetString("fase") == entities.FaseLobby {
			novaEquipe := e.Record.GetString("equipe")
			if novaEquipe == entities.EquipeCorvo || novaEquipe == entities.EquipeAguia {
				jogadoresExistentes, _ := app.FindRecordsByFilter(
					"jogadores",
					"sala_codigo = {:codigo} && equipe = {:eq} && id != {:id}",
					"-created",
					5,
					0,
					dbx.Params{"codigo": salaCodigo, "eq": novaEquipe, "id": e.Record.Id},
				)
				if len(jogadoresExistentes) >= 2 {
					nomeEq := "Corvo"
					if novaEquipe == entities.EquipeAguia {
						nomeEq = "Águia"
					}
					return e.BadRequestError("A Equipe "+nomeEq+" já atingiu o limite de 2 integrantes.", nil)
				}
			}
		}

		return e.Next()
	})

	// 4. Rotas personalizadas (Disputa de PPT, Início de Partida e Snapshot Otimizado)
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Lance de PPT
		se.Router.POST("/api/ppt/lance", func(e *core.RequestEvent) error {
			var req entities.PptMoveRequest
			if err := e.BindBody(&req); err != nil {
				return e.BadRequestError("Formato de payload inválido.", err)
			}

			if req.Rodada <= 0 {
				req.Rodada = 1
			}

			// Verifica se a rodada informada não é obsoleta
			salas, _ := app.FindRecordsByFilter("salas", "codigo = {:codigo}", "-created", 1, 0, dbx.Params{"codigo": req.SalaCodigo})
			if len(salas) > 0 {
				currentRound := salas[0].GetInt("rodada_atual")
				if currentRound > 0 && req.Rodada < currentRound {
					return e.BadRequestError("Rodada obsoleta. A sala já está na rodada "+strconv.Itoa(currentRound), map[string]any{
						"rodada_atual": currentRound,
					})
				}
			}

			res, justResolved, err := pptArbiter.SubmitMove(req)
			if err != nil {
				return e.BadRequestError(err.Error(), nil)
			}

			// Notifica oponente ou atualiza o placar
			if len(salas) > 0 {
				sala := salas[0]
				if justResolved {
					switch res.Vencedor {
					case entities.EquipeCorvo:
						sala.Set("placar_a", sala.GetInt("placar_a")+1)
					case entities.EquipeAguia:
						sala.Set("placar_b", sala.GetInt("placar_b")+1)
					}
					sala.Set("rodada_atual", req.Rodada+1)
					_ = app.Save(sala)
					log.Printf("[PPT] Rodada %d da sala '%s' resolvida: Vencedor %s.", req.Rodada, req.SalaCodigo, res.Vencedor)
				} else {
					// Salva a sala para engatilhar evento SSE para o adversário
					_ = app.Save(sala)
				}
			}

			return e.JSON(200, res)
		})

		// Consulta de status do PPT
		se.Router.GET("/api/ppt/status", func(e *core.RequestEvent) error {
			salaCodigo := e.Request.URL.Query().Get("sala_codigo")
			if salaCodigo == "" {
				return e.BadRequestError("sala_codigo é obrigatório.", nil)
			}

			rodada := 1
			if rStr := e.Request.URL.Query().Get("rodada"); rStr != "" {
				if rNum, err := strconv.Atoi(rStr); err == nil && rNum > 0 {
					rodada = rNum
				}
			}

			status := pptArbiter.GetStatus(salaCodigo, rodada)
			return e.JSON(200, status)
		})

		// Iniciar Partida (Exclusivo do Anfitrião, Quórum estrito de 4 jogadores, balanceamento de aleatórios)
		se.Router.POST("/api/leveling/salas/{codigo}/iniciar", func(e *core.RequestEvent) error {
			codigo := e.Request.PathValue("codigo")
			if codigo == "" {
				return e.BadRequestError("Código da sala é obrigatório.", nil)
			}

			// Identifica o jogador pelo header X-Player-Id ou pelo corpo
			playerId := e.Request.Header.Get("X-Player-Id")
			if playerId == "" {
				type IniciarReq struct {
					PlayerID string `json:"playerId"`
				}
				var req IniciarReq
				_ = e.BindBody(&req)
				playerId = req.PlayerID
			}

			salas, err := app.FindRecordsByFilter("salas", "codigo = {:codigo}", "-created", 1, 0, dbx.Params{"codigo": codigo})
			if err != nil || len(salas) == 0 {
				return e.NotFoundError("Sala não encontrada.", nil)
			}
			sala := salas[0]

			if sala.GetString("fase") != entities.FaseLobby {
				return e.BadRequestError("A partida já foi iniciada.", nil)
			}

			// Valida permissão de anfitrião
			hostId := sala.GetString("host_id")
			if hostId != "" && playerId != "" && hostId != playerId {
				return e.ForbiddenError("Apenas o anfitrião da sala pode iniciar a partida.", nil)
			}

			// Busca todos os jogadores da sala
			jogadores, err := app.FindRecordsByFilter("jogadores", "sala_codigo = {:codigo}", "created", 10, 0, dbx.Params{"codigo": codigo})
			if err != nil || len(jogadores) != 4 {
				return e.BadRequestError("São necessários exatamente 4 jogadores na sala para iniciar a partida.", nil)
			}

			// Se host_id não estava salvo na sala mas o jogador possui is_host, valida
			if hostId == "" {
				isHostFound := false
				for _, jg := range jogadores {
					if jg.GetBool("is_host") && jg.GetString("player_id") == playerId {
						isHostFound = true
						break
					}
				}
				if playerId != "" && !isHostFound && jogadores[0].GetString("player_id") != playerId {
					return e.ForbiddenError("Apenas o anfitrião da sala pode iniciar a partida.", nil)
				}
			}

			// Contagem de equipes e coleta de aleatórios
			var aleatorios []*core.Record
			countA := 0
			countB := 0

			for _, jg := range jogadores {
				eq := jg.GetString("equipe")
				switch eq {
				case entities.EquipeCorvo:
					countA++
				case entities.EquipeAguia:
					countB++
				default:
					aleatorios = append(aleatorios, jg)
				}
			}

			// Se houver participantes indecisos/aleatórios, embaralha estocasticamente
			if len(aleatorios) > 1 {
				for i := len(aleatorios) - 1; i > 0; i-- {
					nBig, errRand := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(i+1)))
					if errRand == nil {
						j := nBig.Int64()
						aleatorios[i], aleatorios[j] = aleatorios[j], aleatorios[i]
					}
				}
			}

			// Aloca os aleatórios para completar 2 na Equipe A e 2 na Equipe B
			for _, jg := range aleatorios {
				if countA < 2 {
					jg.Set("equipe", entities.EquipeCorvo)
					countA++
				} else {
					jg.Set("equipe", entities.EquipeAguia)
					countB++
				}
				if errSave := app.Save(jg); errSave != nil {
					log.Printf("[INICIAR PARTIDA] Erro ao balancear equipe do jogador %s: %v", jg.Id, errSave)
				}
			}

			// Transiciona a sala para INICIATIVA
			sala.Set("fase", entities.FaseIniciativa)
			sala.Set("rodada_atual", 1)
			if errSave := app.Save(sala); errSave != nil {
				return e.InternalServerError("Falha ao atualizar o estado da sala.", errSave)
			}

			log.Printf("[INICIAR PARTIDA] Sala '%s' iniciada pelo host '%s'. Fase: %s (Equipe Corvo: %d, Equipe Águia: %d)",
				codigo, playerId, entities.FaseIniciativa, countA, countB)

			return e.JSON(200, map[string]any{
				"success": true,
				"fase":    entities.FaseIniciativa,
				"message": "Partida iniciada com sucesso.",
			})
		})

		// Rota Otimizada de Snapshot: SQL-Generated JSON (Zero-Copy Pass-Through)
		se.Router.GET("/api/leveling/salas/{codigo}/snapshot", func(e *core.RequestEvent) error {
			codigo := e.Request.PathValue("codigo")
			var rawJSON []byte

			err := app.DB().NewQuery(`
				SELECT json_object(
					'salaCodigo', codigo,
					'fase', fase,
					'hostId', host_id,
					'rodadaAtual', rodada,
					'equipeAtiva', equipe_ativa,
					'placarA', placar_a,
					'placarB', placar_b,
					'vencedor', vencedor,
					'jogadores', (
						SELECT json_group_array(
							json_object(
								'id', id,
								'playerId', player_id,
								'nome', nome,
								'equipe', equipe,
								'papel', papel,
								'isHost', is_host
							)
						)
						FROM jogadores
						WHERE sala_codigo = salas.codigo
					)
				)
				FROM salas
				WHERE codigo = {:codigo}
				LIMIT 1
			`).Bind(dbx.Params{"codigo": codigo}).Row(&rawJSON)

			if err != nil || len(rawJSON) == 0 {
				return e.NotFoundError("Sala não encontrada.", err)
			}

			e.Response.Header().Set("Content-Type", "application/json")
			_, writeErr := e.Response.Write(rawJSON)
			return writeErr
		})

		return se.Next()
	})

	// 4. Agendamento Periódico de Higienização (Zero Bloat)
	app.Cron().MustAdd("cleanupInactivePlayers", "* * * * *", func() {
		cleanup.PurgeInactivePlayers(app, 2)
	})

	app.Cron().MustAdd("cleanupInactiveRooms", "*/5 * * * *", func() {
		cleanup.PurgeInactiveRooms(app, 15)
	})
}
