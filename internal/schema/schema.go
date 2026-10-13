package schema

import (
	_ "embed"
	"log"

	"github.com/pocketbase/pocketbase/core"
)

//go:embed pb_schema.json
var DefaultSchemaJSON []byte

// EnsureCollections assegura que as coleções necessárias estejam criadas no banco de dados
func EnsureCollections(app core.App) error {
	// Verifica se a coleção 'salas' já existe
	_, err := app.FindCollectionByNameOrId("salas")
	if err == nil {
		return nil // Coleções já inicializadas
	}

	log.Println("[BOOTSTRAP] Inicializando coleções do jogo a partir do esquema embarcado...")
	if importErr := app.ImportCollectionsByMarshaledJSON(DefaultSchemaJSON, false); importErr != nil {
		log.Printf("[BOOTSTRAP ERROR] Falha ao importar coleções: %v", importErr)
		return importErr
	}

	log.Println("[BOOTSTRAP] Coleções 'salas', 'rodadas' e 'jogadores' criadas com sucesso!")
	return nil
}
