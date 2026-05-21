package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/honeywire/hub/internal/auth"
	"github.com/honeywire/hub/internal/models"
)

func (h *Handler) ReceiveHeartbeat(w http.ResponseWriter, r *http.Request) {
	var hb models.Heartbeat
	if err := json.NewDecoder(r.Body).Decode(&hb); err != nil {
		RespondError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if hb.SensorID == "" {
		RespondError(w, "sensor_id is required", http.StatusBadRequest)
		return
	}

	nodeID, err := h.authenticateNodeRequest(r)

	if err != nil {
		RespondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)
	minuteBucket := now.Truncate(time.Minute).Format(time.RFC3339)

	// Extract the revision the agent is currently running
	agentRevision := ""
	if rev, ok := hb.Metadata["HW_CONFIG_REV"].(string); ok {
		agentRevision = rev
	}

	// NEW: Marshal the metadata so we can save it to the DB
	metadataJSON, _ := json.Marshal(hb.Metadata)

	// 1. Update Node & Sensor last_seen, Metadata & Reconcile Config
	justSynced, err := h.Store.ProcessHeartbeat(nodeID, hb.SensorID, agentRevision, nowStr, string(metadataJSON))
	if err != nil {
		log.Printf("[ERROR] Heartbeat DB update failed for node %s: %v", nodeID, err)
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}

	// 2. Log heartbeat bucket
	if err := h.Store.InsertHeartbeat(nodeID, hb.SensorID, minuteBucket); err != nil {
		if strings.Contains(err.Error(), "FOREIGN KEY") {
			log.Printf("[INFO] Dropped heartbeat from unregistered sensor %s on node %s (node pending reconciliation)", hb.SensorID, nodeID)
		} else {
			log.Printf("[WARNING] Failed to log heartbeat bucket: %v", err)
		}
	}

	// 3. Broadcasts
	if justSynced {
		h.broadcastWS("NODE_SYNCED", map[string]string{
			"node_id": nodeID,
		})
	}

	h.broadcastWS("SENSOR_HEARTBEAT", map[string]string{
		"node_id":   nodeID,
		"sensor_id": hb.SensorID,
		"timestamp": nowStr,
	})

	SendJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

func (h *Handler) GetUptime(w http.ResponseWriter, r *http.Request) {
	timeframe := r.URL.Query().Get("timeframe")
	if timeframe == "" {
		timeframe = "24H"
	}

	now := time.Now().UTC()
	params := CalculateUptimeParams(timeframe, now)

	sensors, err := h.Store.GetSensorsForUptime(now.Format(time.RFC3339))
	if err != nil {
		RespondError(w, "Database error fetching sensors", http.StatusInternalServerError)
		return
	}

	hbs, err := h.Store.GetHeartbeatsSince(params.CutoffStr)
	if err != nil {
		RespondError(w, "Database error fetching heartbeats", http.StatusInternalServerError)
		return
	}

	result := GenerateUptimeResult(timeframe, now, params, sensors, hbs)
	SendJSON(w, http.StatusOK, result)
}

func (h *Handler) ToggleSilence(w http.ResponseWriter, r *http.Request) {
	// Extract both IDs from the updated URL route
	nodeID := chi.URLParam(r, "id")
	sensorID := chi.URLParam(r, "sensor_id")

	var req struct {
		IsSilenced bool `json:"is_silenced"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	silenceVal := 0
	if req.IsSilenced {
		silenceVal = 1
	}

	if err := h.Store.UpdateSensorSilence(nodeID, sensorID, silenceVal); err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}

	h.broadcastWS("SILENCE_SENSOR", map[string]interface{}{
		"node_id":     nodeID,
		"sensor_id":   sensorID,
		"is_silenced": req.IsSilenced,
	})

	SendJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "success",
		"node_id":     nodeID,
		"sensor_id":   sensorID,
		"is_silenced": req.IsSilenced,
	})
}

// GetManifests fetches the sensor manifest JSON.
func (h *Handler) GetManifests(w http.ResponseWriter, r *http.Request) {
	isAuthenticated := false

	// 1. Try Node API Key Auth
	if r.Header.Get("Authorization") == "" && r.URL.Query().Get("key") != "" {
		r.Header.Set("Authorization", "Bearer "+r.URL.Query().Get("key"))
	}
	_, err := h.authenticateNodeRequest(r)
	if err == nil {
		isAuthenticated = true
	}

	// 2. Try UI Session Auth (Fallback)
	if !isAuthenticated {
		cookie, err := r.Cookie(auth.CookieName)
		if err == nil && cookie.Value != "" {
			if h.SessionStore.IsValid(cookie.Value) {
				isAuthenticated = true
			}
		}
	}

	if !isAuthenticated {
		RespondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := fetchManifestBytes()
	if err != nil {
		RespondError(w, "Failed to reach manifest registry", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}
