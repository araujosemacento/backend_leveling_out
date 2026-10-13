package main

import (
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"leveling-out-backend/internal/hooks"
	"leveling-out-backend/internal/schema"
)

func main() {
	app := pocketbase.New()

	// Garante que as coleções essenciais sejam criadas na primeira execução
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		if err := schema.EnsureCollections(app); err != nil {
			log.Printf("[BOOTSTRAP] Aviso na inicialização de esquemas: %v", err)
		}
		return nil
	})

	// Registra todos os componentes atômicos e hooks autoritativos de Leveling Out
	hooks.RegisterGameHooks(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
