package api

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

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
	Store        *store.SQLiteStore
	Cfg          *config.Config
	SessionStore *auth.SessionStore

	clients       map[*websocket.Conn]bool
	clientsMu     sync.Mutex
	authTracker   map[string]*loginState
	authMutex     sync.Mutex
	nodeAuthCache sync.Map
}

func NewHandler(s *store.SQLiteStore, cfg *config.Config, sess *auth.SessionStore) *Handler {
	h := &Handler{
		Store:        s,
		Cfg:          cfg,
		SessionStore: sess,
		clients:      make(map[*websocket.Conn]bool),
		authTracker:  make(map[string]*loginState),
	}
	go h.cleanupAuthTracker()
	go h.startChartSyncBroadcaster()
	return h
}

// --- Helpers ---

func RespondError(w http.ResponseWriter, message string, code int) {
	SendJSON(w, code, map[string]string{"error": message})
}

func SendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[ERROR] SendJSON encode error: %v\n", err)
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

// Run this in a goroutine when your server starts
func (h *Handler) startChartSyncBroadcaster() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Just tell all connected clients that 30 seconds have passed
		// and they should refresh their time-series charts.
		h.broadcastWS("SYNC_CHARTS", nil)
	}
}

// StartHealthMonitor runs a background task to periodically check for offline nodes and sensors.
// It should be invoked with a context from main.go to ensure graceful shutdown.
func (h *Handler) StartHealthMonitor(ctx context.Context) {
	log.Println("[INFO] Starting background health monitor...")

	tickerPeriod := 30 * time.Second
	ticker := time.NewTicker(tickerPeriod)
	defer ticker.Stop()

	// Offset lastCheck by the ticker period so the first run catches recent drops
	lastCheck := time.Now().UTC().Add(-tickerPeriod)

	for {
		select {
		case <-ctx.Done():
			log.Println("[INFO] Health monitor stopped")
			return
		case t := <-ticker.C:
			offlineThreshold := 60 * time.Second
			updatedNodeIDs, err := h.Store.GetTransitionedOfflineNodes(offlineThreshold, lastCheck)

			if err == nil {
				for nodeID := range updatedNodeIDs {
					// Send a lightweight signal to instruct the UI to securely refresh this node's state
					h.broadcastWS("UPDATE_NODE", map[string]interface{}{
						"id":              nodeID,
						"trigger_refresh": true,
					})
				}
			}

			lastCheck = t.UTC()
		}
	}
}
