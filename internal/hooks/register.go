package hooks

import (
	"github.com/pocketbase/pocketbase/core"

	"leveling-out-backend/internal/components"
)

// RegisterGameHooks instancia os componentes atômicos autoritativos e orquestra o registro de hooks, rotas e crons
func RegisterGameHooks(app core.App) {
	proximity := components.NewProximityComponent()
	secretTarget := components.NewSecretTargetComponent()
	pptArbiter := components.NewPptArbiterComponent()
	cleanup := components.NewCleanupComponent()

	// 1. Hooks de ciclo de vida das coleções salas, rodadas e jogadores
	registerLifecycleHooks(app, proximity, secretTarget)

	// 2. Rotas HTTP autoritativas (Lobby, PPT e Partida)
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		registerLobbyRoutes(se, app)
		registerPptRoutes(se, app, pptArbiter)
		registerMatchRoutes(se, app)
		return se.Next()
	})

	// 3. Agendamento periódico de higienização
	registerCronJobs(app, cleanup)
}
