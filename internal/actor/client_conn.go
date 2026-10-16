package actor

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// ClientConn encapsula uma conexão WebSocket ativa de um jogador
type ClientConn struct {
	PlayerID     string
	DeviceID     string
	Nome         string
	SessionToken string
	ws           *websocket.Conn
	sendChan     chan []byte
	mu           sync.Mutex
	fechado      bool
}

// NewClientConn instancia o wrapper de conexão com buffer de saída
func NewClientConn(ws *websocket.Conn, playerID, deviceID, nome, token string) *ClientConn {
	return &ClientConn{
		PlayerID:     playerID,
		DeviceID:     deviceID,
		Nome:         nome,
		SessionToken: token,
		ws:           ws,
		sendChan:     make(chan []byte, 64),
	}
}

// EnviarJSON serializa e enfileira uma mensagem para envio no socket
func (c *ClientConn) EnviarJSON(v interface{}) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fechado {
		return false
	}

	data, err := json.Marshal(v)
	if err != nil {
		return false
	}

	select {
	case c.sendChan <- data:
		return true
	default:
		// Buffer cheio: descarta frame para não travar goroutines
		return false
	}
}

// IniciarLoopEscrita processa a fila de saída do socket
func (c *ClientConn) IniciarLoopEscrita(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-c.sendChan:
			if !ok {
				return
			}
			writeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			err := c.ws.Write(writeCtx, websocket.MessageText, data)
			cancel()
			if err != nil {
				c.Fechar()
				return
			}
		}
	}
}

// EnviarJSONESair serializa o payload, escreve diretamente no socket com timeout curto e encerra a conexão
func (c *ClientConn) EnviarJSONESair(v interface{}, code websocket.StatusCode, reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fechado {
		return
	}
	c.fechado = true

	data, err := json.Marshal(v)
	if err == nil && c.ws != nil {
		writeCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_ = c.ws.Write(writeCtx, websocket.MessageText, data)
		cancel()
	}

	close(c.sendChan)
	if c.ws != nil {
		_ = c.ws.Close(code, reason)
	}
}

// FecharComStatus encerra o socket com status e motivo especificados
func (c *ClientConn) FecharComStatus(code websocket.StatusCode, reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.fechado {
		c.fechado = true
		close(c.sendChan)
		if c.ws != nil {
			c.ws.Close(code, reason)
		}
	}
}

// Fechar encerra o socket com fechamento normal
func (c *ClientConn) Fechar() {
	c.FecharComStatus(websocket.StatusNormalClosure, "conexão encerrada")
}

