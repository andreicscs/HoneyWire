package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/honeywire/hub/internal/auth"
	"github.com/honeywire/hub/internal/config"
	"github.com/honeywire/hub/internal/store"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Handler struct {
	Store        *store.Store
	Cfg          *config.Config
	SessionStore *auth.SessionStore

	clients   map[*websocket.Conn]bool
	clientsMu sync.Mutex
}

func NewHandler(s *store.Store, cfg *config.Config, sess *auth.SessionStore) *Handler {
	h := &Handler{
		Store:        s,
		Cfg:          cfg,
		SessionStore: sess,
		clients:      make(map[*websocket.Conn]bool),
	}
	go h.cleanupAuthTracker()
	return h
}

// --- Helpers ---

func SendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("SendJSON encode error: %v\n", err)
	}
}

func (h *Handler) getRealIP(r *http.Request) string {
	if h.Cfg.TrustProxy {
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			return strings.Split(ip, ",")[0]
		}
		if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
			return strings.Split(ip, ",")[0]
		}
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// --- WebSocket ---

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	h.clientsMu.Lock()
	h.clients[conn] = true
	h.clientsMu.Unlock()

	go func() {
		defer func() {
			h.clientsMu.Lock()
			delete(h.clients, conn)
			h.clientsMu.Unlock()
			conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

func (h *Handler) broadcastWS(msgType string, payload interface{}) {
	var deadClients []*websocket.Conn
	h.clientsMu.Lock()
	for client := range h.clients {
		err := client.WriteJSON(map[string]interface{}{
			"type":    msgType,
			"payload": payload,
		})
		if err != nil {
			deadClients = append(deadClients, client)
		}
	}
	for _, c := range deadClients {
		delete(h.clients, c)
		c.Close()
	}
	h.clientsMu.Unlock()
}