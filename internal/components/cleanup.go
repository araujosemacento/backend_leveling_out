package components

import (
	"log"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// CleanupComponent encapsula as rotinas de expurgo automático de recursos inativos
type CleanupComponent struct{}

// NewCleanupComponent inicializa o componente de higienização
func NewCleanupComponent() *CleanupComponent {
	return &CleanupComponent{}
}

// PurgeInactivePlayers remove registros de jogadores sem ping recente (last_seen > minutos)
func (c *CleanupComponent) PurgeInactivePlayers(app core.App, minutes int) {
	threshold := time.Now().UTC().Add(-time.Duration(minutes) * time.Minute).Format("2006-01-02 15:04:05.000Z")

	players, err := app.FindRecordsByFilter(
		"jogadores",
		"last_seen < {:threshold}",
		"-last_seen",
		0,
		0,
		dbx.Params{"threshold": threshold},
	)
	if err != nil {
		log.Printf("[CLEANUP ERROR] Falha ao consultar jogadores inativos: %v", err)
		return
	}

	if len(players) > 0 {
		purgados := 0
		for _, p := range players {
			salaCodigo := p.GetString("sala_codigo")
			salas, errSala := app.FindAllRecords("salas", dbx.HashExp{"codigo": salaCodigo})
			// Só purga se a sala não existe mais ou se ainda estiver no LOBBY / já FIM_JOGO
			if errSala == nil && len(salas) > 0 {
				fase := salas[0].GetString("fase")
				if fase != "LOBBY" && fase != "FIM_JOGO" {
					continue
				}
			}

			if err := app.Delete(p); err != nil {
				log.Printf("[CLEANUP ERROR] Erro ao deletar jogador '%s': %v", p.Id, err)
			} else {
				purgados++
			}
		}
		if purgados > 0 {
			log.Printf("[CLEANUP] %d jogador(es) inativo(s) purgado(s) com sucesso.", purgados)
		}
	}
}

// PurgeInactiveRooms remove em cascata salas sem jogadores ativos recentes (last_seen > minutos)
func (c *CleanupComponent) PurgeInactiveRooms(app core.App, minutes int) {
	threshold := time.Now().UTC().Add(-time.Duration(minutes) * time.Minute).Format("2006-01-02 15:04:05.000Z")

	var staleRoomCodes []string
	err := app.DB().NewQuery(`
		SELECT codigo FROM salas
		WHERE codigo NOT IN (
			SELECT DISTINCT sala_codigo FROM jogadores WHERE last_seen >= {:threshold}
		)
	`).Bind(dbx.Params{"threshold": threshold}).Column(&staleRoomCodes)

	if err != nil {
		log.Printf("[CLEANUP ERROR] Falha ao consultar salas inativas: %v", err)
		return
	}

	if len(staleRoomCodes) > 0 {
		log.Printf("[CLEANUP] Iniciando higienização: %d sala(s) inativa(s) detectada(s).", len(staleRoomCodes))

		for _, codigo := range staleRoomCodes {
			rooms, rErr := app.FindAllRecords("salas", dbx.HashExp{"codigo": codigo})
			if rErr == nil && len(rooms) > 0 {
				room := rooms[0]
				salaCodigo := room.GetString("codigo")

				// Remove jogadores vinculados
				players, pErr := app.FindAllRecords("jogadores", dbx.HashExp{"sala_codigo": salaCodigo})
				if pErr == nil {
					for _, p := range players {
						_ = app.Delete(p)
					}
				}

				// Remove rodadas vinculadas
				rounds, rErr := app.FindAllRecords("rodadas", dbx.HashExp{"sala_codigo": salaCodigo})
				if rErr == nil {
					for _, r := range rounds {
						_ = app.Delete(r)
					}
				}

				// Remove o registro principal da sala
				if err := app.Delete(room); err == nil {
					log.Printf("[CLEANUP] Sala '%s', participantes e histórico de rodadas purgados.", salaCodigo)
				}
			}
		}

		log.Println("[CLEANUP] Higienização de salas concluída com sucesso. Espaço liberado.")
	}
}
