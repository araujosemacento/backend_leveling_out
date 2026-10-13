package hooks

import (
	"log"
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

	// 3. Rotas personalizadas (Disputa de PPT e Snapshot Otimizado)
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

		// Rota Otimizada de Snapshot: SQL-Generated JSON (Zero-Copy Pass-Through)
		se.Router.GET("/api/leveling/salas/{codigo}/snapshot", func(e *core.RequestEvent) error {
			codigo := e.Request.PathValue("codigo")
			var rawJSON []byte

			err := app.DB().NewQuery(`
				SELECT json_object(
					'salaCodigo', codigo,
					'fase', fase,
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
								'papel', papel
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
