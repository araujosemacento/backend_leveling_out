package hooks

import (
	cryptorand "crypto/rand"
	"log"
	"math/big"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"leveling-out-backend/internal/entities"
)

// registerMatchRoutes registra as rotas HTTP de controle e snapshot da partida
func registerMatchRoutes(se *core.ServeEvent, app core.App, ppt *pptService) {
	// Iniciar Partida (Exclusivo do Anfitrião, Quórum estrito de 4 jogadores, balanceamento de aleatórios)
	se.Router.POST("/api/leveling/salas/{codigo}/iniciar", func(e *core.RequestEvent) error {
		codigo := e.Request.PathValue("codigo")
		if codigo == "" {
			return e.BadRequestError("Código da sala é obrigatório.", nil)
		}

		// Identifica o jogador pelo header X-Player-Id ou pelo corpo
		playerId := e.Request.Header.Get("X-Player-Id")
		if playerId == "" {
			type IniciarReq struct {
				PlayerID string `json:"playerId"`
			}
			var req IniciarReq
			_ = e.BindBody(&req)
			playerId = req.PlayerID
		}

		salas, err := app.FindAllRecords("salas", dbx.HashExp{"codigo": codigo})
		if err != nil || len(salas) == 0 {
			return e.NotFoundError("Sala não encontrada.", nil)
		}
		sala := salas[0]

		if sala.GetString("fase") != entities.FaseLobby {
			return e.BadRequestError("A partida já foi iniciada.", nil)
		}

		// Valida permissão de anfitrião
		hostId := sala.GetString("host_id")
		if hostId != "" && playerId != "" && hostId != playerId {
			return e.ForbiddenError("Apenas o anfitrião da sala pode iniciar a partida.", nil)
		}

		// Busca todos os jogadores da sala
		jogadores, err := app.FindAllRecords("jogadores", dbx.HashExp{"sala_codigo": codigo})
		if err != nil || len(jogadores) != 4 {
			return e.BadRequestError("São necessários exatamente 4 jogadores na sala para iniciar a partida.", nil)
		}

		// Se host_id não estava salvo na sala mas o jogador possui is_host, valida
		if hostId == "" {
			isHostFound := false
			for _, jg := range jogadores {
				if jg.GetBool("is_host") && jg.GetString("player_id") == playerId {
					isHostFound = true
					break
				}
			}
			if playerId != "" && !isHostFound && jogadores[0].GetString("player_id") != playerId {
				return e.ForbiddenError("Apenas o anfitrião da sala pode iniciar a partida.", nil)
			}
		}

		// Contagem de equipes e coleta de manuais e aleatórios
		var manuaisA []*core.Record
		var manuaisB []*core.Record
		var aleatorios []*core.Record

		for _, jg := range jogadores {
			eq := jg.GetString("equipe")
			switch eq {
			case entities.EquipePato:
				manuaisA = append(manuaisA, jg)
			case entities.EquipeGanso:
				manuaisB = append(manuaisB, jg)
			default:
				aleatorios = append(aleatorios, jg)
			}
		}

		// Se houver participantes indecisos/aleatórios, embaralha estocasticamente
		if len(aleatorios) > 1 {
			for i := len(aleatorios) - 1; i > 0; i-- {
				nBig, errRand := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(i+1)))
				if errRand == nil {
					j := nBig.Int64()
					aleatorios[i], aleatorios[j] = aleatorios[j], aleatorios[i]
				}
			}
		}

		countA := len(manuaisA)
		countB := len(manuaisB)
		var sorteadosA []*core.Record
		var sorteadosB []*core.Record
		now := time.Now().UTC()

		// Aloca os aleatórios para completar 2 na Equipe A e 2 na Equipe B
		for i, jg := range aleatorios {
			tSorteio := now.Add(time.Duration(i) * time.Millisecond).Format(formatoDataPB)
			jg.Set("equipe_desde", tSorteio)
			if countA < 2 {
				jg.Set("equipe", entities.EquipePato)
				countA++
				sorteadosA = append(sorteadosA, jg)
			} else {
				jg.Set("equipe", entities.EquipeGanso)
				countB++
				sorteadosB = append(sorteadosB, jg)
			}
			if errSave := app.Save(jg); errSave != nil {
				log.Printf("[INICIAR PARTIDA] Erro ao balancear equipe do jogador %s: %v", jg.Id, errSave)
			}
		}

		// Determina os representantes oficiais de cada equipe para o minijogo PPT
		repA := escolherRepresentante(manuaisA, sorteadosA)
		repB := escolherRepresentante(manuaisB, sorteadosB)

		// Inicializa a disputa de iniciativa no serviço PPT
		ppt.Iniciar(sala, repA, repB)

		// Transiciona a sala para INICIATIVA
		sala.Set("fase", entities.FaseIniciativa)
		sala.Set("rodada", 1)
		sala.Set("rodada_atual", 1)
		if errSave := app.Save(sala); errSave != nil {
			return e.InternalServerError("Falha ao atualizar o estado da sala.", errSave)
		}

		log.Printf("[INICIAR PARTIDA] Sala '%s' iniciada pelo host '%s'. Fase: %s (Pato: %s, Ganso: %s)",
			codigo, playerId, entities.FaseIniciativa, repA, repB)

		return e.JSON(200, map[string]any{
			"success": true,
			"fase":    entities.FaseIniciativa,
			"message": "Partida iniciada com sucesso.",
		})
	})

	// Rota Otimizada de Snapshot: SQL-Generated JSON (Zero-Copy Pass-Through)
	se.Router.GET("/api/leveling/salas/{codigo}/snapshot", func(e *core.RequestEvent) error {
		codigo := e.Request.PathValue("codigo")
		rawJSON, err := fetchRoomSnapshotRaw(app, codigo)
		if err != nil || len(rawJSON) == 0 {
			return e.NotFoundError("Sala não encontrada.", err)
		}

		e.Response.Header().Set("Content-Type", "application/json")
		_, writeErr := e.Response.Write(rawJSON)
		return writeErr
	})
}
