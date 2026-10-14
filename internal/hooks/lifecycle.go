package hooks

import (
	"log"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"leveling-out-backend/internal/components"
	"leveling-out-backend/internal/entities"
)

// registerLifecycleHooks registra as regras de negócio disparadas no ciclo de vida das coleções salas, rodadas e jogadores
func registerLifecycleHooks(app core.App, proximity *components.ProximityComponent, secretTarget *components.SecretTargetComponent) {
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

	// 3. Atualização e progressão da rodada: validações, cálculo de pontos e atualização em cascata
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

			salas, err := app.FindAllRecords("salas", dbx.HashExp{"codigo": salaCodigo})
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

	// 4. Regras autoritativas para a coleção jogadores (Quórum máximo de 4, controle de anfitrião e lotação de equipes)
	app.OnRecordCreateRequest("jogadores").BindFunc(func(e *core.RecordRequestEvent) error {
		salaCodigo := e.Record.GetString("sala_codigo")
		if salaCodigo == "" {
			return e.BadRequestError("sala_codigo é obrigatório.", nil)
		}

		jogadoresExistentes, _ := app.FindAllRecords("jogadores", dbx.HashExp{"sala_codigo": salaCodigo})

		if len(jogadoresExistentes) >= 4 {
			return e.BadRequestError("A sala já atingiu a capacidade máxima de 4 jogadores.", nil)
		}

		// O primeiro jogador que entrar na sala torna-se automaticamente o dono (host)
		if len(jogadoresExistentes) == 0 {
			e.Record.Set("is_host", true)
			salas, _ := app.FindAllRecords("salas", dbx.HashExp{"codigo": salaCodigo})
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
		salas, _ := app.FindAllRecords("salas", dbx.HashExp{"codigo": salaCodigo})
		if len(salas) > 0 && salas[0].GetString("fase") == entities.FaseLobby {
			novaEquipe := e.Record.GetString("equipe")
			if novaEquipe == entities.EquipeCorvo || novaEquipe == entities.EquipeAguia {
				jogadoresExistentes, _ := app.FindAllRecords(
					"jogadores",
					dbx.HashExp{"sala_codigo": salaCodigo, "equipe": novaEquipe},
					dbx.NewExp("id != {:id}", dbx.Params{"id": e.Record.Id}),
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
}
