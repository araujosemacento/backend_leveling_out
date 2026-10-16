package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/coder/websocket"

	"leveling-out-backend/internal/actor"
	"leveling-out-backend/internal/hub"
	"leveling-out-backend/internal/protocol"
)

func main() {
	flags := flag.NewFlagSet("leveling-out-server", flag.ExitOnError)
	httpAddr := flags.String("http", "127.0.0.1:8090", "Endereço de escuta do servidor HTTP/WebSocket")

	args := os.Args[1:]
	if len(args) > 0 && args[0] == "serve" {
		args = args[1:]
	}
	_ = flags.Parse(args)

	if envPort := os.Getenv("PORT"); envPort != "" {
		*httpAddr = "0.0.0.0:" + envPort
	}

	proxySecret := os.Getenv("PROXY_SECRET")
	if proxySecret == "" {
		proxySecret = os.Getenv("PUBLIC_PB_PROXY_SECRET")
	}

	lobbyHub := hub.NewLobbyHub()
	mux := http.NewServeMux()

	// Healthcheck simples
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "ok",
			"salasAtivas": lobbyHub.SalasAtivas(),
		})
	})

	// Endpoint WebSocket autoritativo por sala: /ws/salas/{codigo}
	mux.HandleFunc("/ws/salas/{codigo}", func(w http.ResponseWriter, r *http.Request) {
		codigo := r.PathValue("codigo")
		if codigo == "" {
			http.Error(w, "Código de sala obrigatório", http.StatusBadRequest)
			return
		}

		// Validação defensiva do Proxy Secret (se configurado no ambiente)
		if proxySecret != "" {
			reqSecret := r.URL.Query().Get("x-proxy-secret")
			if reqSecret == "" {
				reqSecret = r.Header.Get("X-Proxy-Secret")
			}
			if reqSecret != proxySecret {
				http.Error(w, "Acesso não autorizado pelo proxy", http.StatusForbidden)
				return
			}
		}

		// Parâmetros de identificação do participante
		playerID := strings.TrimSpace(r.URL.Query().Get("player_id"))
		if playerID == "" {
			http.Error(w, "Parâmetro player_id obrigatório", http.StatusBadRequest)
			return
		}
		deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
		token := strings.TrimSpace(r.URL.Query().Get("token"))
		nome := strings.TrimSpace(r.URL.Query().Get("nome"))
		if nome == "" {
			nome = "Convidado"
		}

		// Upgrade HTTP -> WebSocket (aceita origens cruzadas locais e de produção)
		ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: []string{"*"},
		})
		if err != nil {
			log.Printf("[WS] Falha no upgrade WebSocket (%s): %v", codigo, err)
			return
		}

		clientConn := actor.NewClientConn(ws, playerID, deviceID, nome, token)
		sala := lobbyHub.ObterOuCriar(codigo)

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		// Goroutine de escrita do cliente
		go clientConn.IniciarLoopEscrita(ctx)

		// Notifica o Actor sobre a nova conexão
		sala.Despachar("CONECTAR", playerID, nil, clientConn)

		// Loop de leitura bloqueante nesta goroutine
		for {
			_, msgBytes, err := ws.Read(ctx)
			if err != nil {
				// Queda de socket ou fechamento de aba
				break
			}

			var msg protocol.MensagemIn
			if err := json.Unmarshal(msgBytes, &msg); err == nil && msg.Tipo != "" {
				sala.Despachar(msg.Tipo, playerID, msg.Payload, clientConn)
			}
		}

		// Limpeza e notificação de desconexão instantânea
		clientConn.Fechar()
		sala.Despachar("DESCONECTAR", playerID, nil, nil)
	})

	server := &http.Server{
		Addr:         *httpAddr,
		Handler:      corsMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Leveling Out Game Server (Go Nativo In-Memory) iniciado em http://%s", *httpAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[SERVER] Falha crítica de execução: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[SERVER] Encerrando servidor e liberando conexões...")
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelShutdown()
	_ = server.Shutdown(ctxShutdown)
	log.Println("[SERVER] Servidor finalizado com sucesso.")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Player-Id, X-Session-Token, X-Proxy-Secret")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
