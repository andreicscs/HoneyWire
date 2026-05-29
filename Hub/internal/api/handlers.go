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

	"github.com/honeywire/hub/internal/auth"
	"github.com/honeywire/hub/internal/config"
	"github.com/honeywire/hub/internal/services/node"
	"github.com/honeywire/hub/internal/services/sensor"
	"github.com/honeywire/hub/internal/services/websocket"
	"github.com/honeywire/hub/internal/store"
)

type Handler struct {
	Store         *store.SQLiteStore
	Cfg           *config.Config
	SessionStore  *auth.SessionStore
	WSService     *websocket.Service
	NodeService   *node.Service
	SensorService *sensor.Service

	authTracker   map[string]*loginState
	authMutex     sync.Mutex
	nodeAuthCache sync.Map
}

func NewHandler(s *store.SQLiteStore, cfg *config.Config, sess *auth.SessionStore, ws *websocket.Service, nodeSvc *node.Service, sensorSvc *sensor.Service) *Handler {
	h := &Handler{
		Store:         s,
		Cfg:           cfg,
		SessionStore:  sess,
		WSService:     ws,
		NodeService:   nodeSvc,
		SensorService: sensorSvc,
		authTracker:   make(map[string]*loginState),
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

// TODO Legacy helper to bridge older API endpoints until they are moved to Domain Services
func (h *Handler) broadcastWS(msgType string, payload interface{}) {
	h.WSService.Broadcast(msgType, payload)
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
