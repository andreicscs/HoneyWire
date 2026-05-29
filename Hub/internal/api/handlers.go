package api

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/honeywire/hub/internal/services/websocket"
	"github.com/honeywire/hub/internal/store"
)

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

func GetRealIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
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

// StartHealthMonitor runs a background task to periodically check for offline nodes and sensors.
// It should be invoked with a context from main.go to ensure graceful shutdown.
func StartHealthMonitor(ctx context.Context, s *store.SQLiteStore, ws *websocket.Service) {
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
			updatedNodeIDs, err := s.GetTransitionedOfflineNodes(offlineThreshold, lastCheck)

			if err == nil {
				for nodeID := range updatedNodeIDs {
					// Send a lightweight signal to instruct the UI to securely refresh this node's state
					ws.Broadcast("UPDATE_NODE", map[string]interface{}{
						"id":              nodeID,
						"trigger_refresh": true,
					})
				}
			}

			lastCheck = t.UTC()
		}
	}
}
