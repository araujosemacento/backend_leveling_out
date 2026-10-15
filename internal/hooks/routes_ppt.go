package hooks

import (
	"log"
	"strconv"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"leveling-out-backend/internal/components"
	"leveling-out-backend/internal/entities"
)

// registerPptRoutes registra as rotas HTTP de disputa e consulta de Pedra, Papel e Tesoura
func registerPptRoutes(se *core.ServeEvent, app core.App, pptArbiter *components.PptArbiterComponent) {
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
		salas, _ := app.FindAllRecords("salas", dbx.HashExp{"codigo": req.SalaCodigo})
		if len(salas) > 0 {
			currentRound := salas[0].GetInt("rodada")
			if currentRound <= 0 {
				currentRound = salas[0].GetInt("rodada_atual")
			}
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
				case entities.EquipePato:
					sala.Set("placar_a", sala.GetInt("placar_a")+1)
				case entities.EquipeGanso:
					sala.Set("placar_b", sala.GetInt("placar_b")+1)
				}
				sala.Set("rodada", req.Rodada+1)
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
}
