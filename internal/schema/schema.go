package schema

import (
	_ "embed"
	"log"

	"github.com/pocketbase/pocketbase/core"
)

//go:embed pb_schema.json
var DefaultSchemaJSON []byte

// EnsureCollections assegura que as coleções necessárias estejam criadas e atualizadas no banco de dados
func EnsureCollections(app core.App) error {
	log.Println("[BOOTSTRAP] Sincronizando coleções do jogo a partir do esquema embarcado...")
	if importErr := app.ImportCollectionsByMarshaledJSON(DefaultSchemaJSON, false); importErr != nil {
		log.Printf("[BOOTSTRAP ERROR] Falha ao sincronizar coleções: %v", importErr)
		return importErr
	}

	log.Println("[BOOTSTRAP] Coleções 'salas', 'rodadas' e 'jogadores' sincronizadas com sucesso!")
	return nil
}
