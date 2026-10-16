package hooks

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// generateSecureToken gera uma string hexadecimal aleatória com entropia criptográfica (32 bytes)
func generateSecureToken() string {
	b := make([]byte, 32)
	if _, err := cryptorand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b)
}

// sanitizeNome limpa e normaliza o apelido do jogador defensivamente
func sanitizeNome(nome string) string {
	nome = strings.TrimSpace(nome)
	nome = strings.ReplaceAll(nome, "<", "")
	nome = strings.ReplaceAll(nome, ">", "")
	if len(nome) > 20 {
		nome = nome[:20]
	}
	return nome
}

// fetchRoomSnapshotRaw executa a consulta otimizada SQL-Generated JSON (Pass-Through)
func fetchRoomSnapshotRaw(app core.App, codigo string) ([]byte, error) {
	var rawJSON []byte
	err := app.DB().NewQuery(`
		SELECT json_object(
			'salaCodigo', codigo,
			'fase', fase,
			'hostId', host_id,
			'rodadaAtual', rodada,
			'equipeAtiva', equipe_ativa,
			'placarA', placar_a,
			'placarB', placar_b,
			'vencedor', vencedor,
			'motivoFim', motivo_fim,
			'pptEstado', ppt_estado,
			'pptPrazo', ppt_prazo,
			'pptRevelacaoEm', ppt_revelacao_em,
			'pptEnviouA', ppt_enviou_a,
			'pptEnviouB', ppt_enviou_b,
			'pptRepresentanteA', ppt_representante_a,
			'pptRepresentanteB', ppt_representante_b,
			'pptResultado', ppt_resultado,
			'pptLanceA', ppt_lance_a,
			'pptLanceB', ppt_lance_b,
			'jogadores', (
				SELECT json_group_array(
					json_object(
						'id', id,
						'playerId', player_id,
						'nome', nome,
						'equipe', equipe,
						'papel', papel,
						'isHost', is_host,
						'equipeDesde', equipe_desde
					)
				)
				FROM jogadores
				WHERE sala_codigo = salas.codigo
			)
		)
		FROM salas
		WHERE codigo = {:codigo}
		LIMIT 1
	`).Bind(dbx.Params{"codigo": codigo}).Row(&rawJSON)

	if err != nil || len(rawJSON) == 0 {
		return nil, err
	}
	return rawJSON, nil
}
