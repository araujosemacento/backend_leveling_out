package hooks

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"leveling-out-backend/internal/entities"
)

// autenticarJogador valida estritamente o par X-Player-Id / X-Session-Token do participante da sala
func autenticarJogador(app core.App, codigo string, e *core.RequestEvent) (*core.Record, error) {
	playerID := e.Request.Header.Get("X-Player-Id")
	token := e.Request.Header.Get("X-Session-Token")
	if codigo == "" || playerID == "" || token == "" {
		return nil, e.UnauthorizedError("Credenciais de sessão obrigatórias.", nil)
	}
	jogadores, err := app.FindAllRecords("jogadores", dbx.HashExp{"sala_codigo": codigo, "player_id": playerID})
	if err != nil || len(jogadores) == 0 {
		return nil, e.NotFoundError("Jogador não encontrado.", nil)
	}
	if jogadores[0].GetString("session_token") != token {
		return nil, e.UnauthorizedError("Token de sessão inválido.", nil)
	}
	return jogadores[0], nil
}

// registerPptRoutes registra as rotas autenticadas da disputa de Pedra, Papel e Tesoura
func registerPptRoutes(se *core.ServeEvent, app core.App, svc *pptService) {
	// Lance do representante (definitivo; equipe derivada do registro do jogador)
	se.Router.POST("/api/leveling/salas/{codigo}/ppt/lance", func(e *core.RequestEvent) error {
		codigo := e.Request.PathValue("codigo")
		jogador, err := autenticarJogador(app, codigo, e)
		if err != nil {
			return err
		}

		var req entities.PptMoveRequest
		if err := e.BindBody(&req); err != nil {
			return e.BadRequestError("Formato de payload inválido.", err)
		}

		if herr := svc.Enviar(codigo, jogador, req.Lance); herr != nil {
			return e.JSON(herr.Status, map[string]any{"message": herr.Message})
		}
		return e.JSON(200, map[string]any{"success": true})
	})

	// Lance sigiloso da própria equipe (para que o colega espectador veja "Sua equipe escolheu ...")
	se.Router.GET("/api/leveling/salas/{codigo}/ppt/meu-lance", func(e *core.RequestEvent) error {
		codigo := e.Request.PathValue("codigo")
		jogador, err := autenticarJogador(app, codigo, e)
		if err != nil {
			return err
		}
		return e.JSON(200, map[string]any{"lance": svc.LanceDaEquipe(codigo, jogador)})
	})
}
