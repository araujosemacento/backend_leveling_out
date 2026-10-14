package hooks

import (
	"github.com/pocketbase/pocketbase/core"

	"leveling-out-backend/internal/components"
)

// registerCronJobs agenda os trabalhos de limpeza periódica de jogadores e salas inativos
func registerCronJobs(app core.App, cleanup *components.CleanupComponent) {
	app.Cron().MustAdd("cleanupInactivePlayers", "* * * * *", func() {
		cleanup.PurgeInactivePlayers(app, 2)
	})

	app.Cron().MustAdd("cleanupInactiveRooms", "*/5 * * * *", func() {
		cleanup.PurgeInactiveRooms(app, 15)
	})
}
