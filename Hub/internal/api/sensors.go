package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/honeywire/hub/internal/models"
)

func (h *Handler) ReceiveHeartbeat(w http.ResponseWriter, r *http.Request) {
	var hb models.Heartbeat
	if err := json.NewDecoder(r.Body).Decode(&hb); err != nil {
		RespondError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if hb.NodeID == "" || hb.SensorID == "" {
		RespondError(w, "node_id and sensor_id are required", http.StatusBadRequest)
		return
	}

	// Per-node authentication
	if !h.validateNodeAuth(r, hb.NodeID) {
		RespondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)
	minuteBucket := now.Truncate(time.Minute).Format(time.RFC3339)
	metadataJSON, _ := json.Marshal(hb.Metadata)

	// Check if this is a new sensor (first heartbeat)
	existingSensor, _ := h.Store.GetSensor(hb.NodeID, hb.SensorID)
	isNewSensor := existingSensor == nil

	// Update sensor last_seen and metadata with composite key (node_id, sensor_id)
	if err := h.Store.UpsertSensor(&hb, nowStr, string(metadataJSON)); err != nil {
		log.Printf("[ERROR] Heartbeat DB Upsert failed for node %s/sensor %s: %v", hb.NodeID, hb.SensorID, err)
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Update node last_seen
	if err := h.Store.UpdateNodeLastSeen(hb.NodeID, nowStr); err != nil {
		log.Printf("[WARNING] Failed to update node %s last_seen: %v", hb.NodeID, err)
	}

	// Log heartbeat bucket with composite key
	if err := h.Store.InsertHeartbeat(hb.NodeID, hb.SensorID, minuteBucket); err != nil {
		log.Printf("[WARNING] Failed to log heartbeat bucket for node %s/sensor %s: %v", hb.NodeID, hb.SensorID, err)
	}

	// Broadcast NEW_SENSOR only on first heartbeat, then SENSOR_HEARTBEAT every time
	if isNewSensor {
		h.broadcastWS("NEW_SENSOR", map[string]string{
			"node_id":   hb.NodeID,
			"sensor_id": hb.SensorID,
			"timestamp": nowStr,
		})
	} else {
		h.broadcastWS("SENSOR_HEARTBEAT", map[string]string{
			"node_id":   hb.NodeID,
			"sensor_id": hb.SensorID,
			"timestamp": nowStr,
		})
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

func (h *Handler) GetSensors(w http.ResponseWriter, r *http.Request) {
	fleet, err := h.Store.GetAllSensors()
	if err != nil {
		RespondError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, fleet)
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
	sensorID := chi.URLParam(r, "sensor_id")
	
	// Add NodeID to the expected JSON payload
	var req struct {
		NodeID     string `json:"node_id"`
		IsSilenced bool   `json:"is_silenced"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.NodeID == "" {
		RespondError(w, "node_id is required", http.StatusBadRequest)
		return
	}

	silenceVal := 0
	if req.IsSilenced {
		silenceVal = 1
	}

	if err := h.Store.UpdateSensorSilence(req.NodeID, sensorID, silenceVal); err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}

	h.broadcastWS("SILENCE_SENSOR", map[string]interface{}{
		"node_id":     req.NodeID,
		"sensor_id":   sensorID,
		"is_silenced": req.IsSilenced,
	})

	SendJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "success",
		"node_id":     req.NodeID,
		"sensor_id":   sensorID,
		"is_silenced": req.IsSilenced,
	})
}

func (h *Handler) ForgetSensor(w http.ResponseWriter, r *http.Request) {
	sensorID := chi.URLParam(r, "sensor_id")
	// Use a URL query parameter for DELETE requests (e.g., ?node_id=1234)
	nodeID := r.URL.Query().Get("node_id")

	if nodeID == "" {
		RespondError(w, "node_id query parameter is required", http.StatusBadRequest)
		return
	}

	rowsAffected, err := h.Store.DeleteSensor(nodeID, sensorID)
	if err != nil {
		RespondError(w, "Database error while deleting sensor", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		RespondError(w, "Sensor not found", http.StatusNotFound)
		return
	}

	h.broadcastWS("DELETE_SENSOR", map[string]string{
		"node_id":   nodeID,
		"sensor_id": sensorID,
	})

	SendJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Sensor forgotten successfully",
	})
}

// GetManifests fetches the sensor manifest JSON.
// It proxies the request through the Hub to bypass browser CORS restrictions.
func (h *Handler) GetManifests(w http.ResponseWriter, r *http.Request) {
	// 1. Determine the target URL
	manifestURL := os.Getenv("HW_MANIFEST_URL")
	if manifestURL == "" {
		// Production fallback
		manifestURL = "https://raw.githubusercontent.com/andreicscs/HoneyWire/main/Sensors/official/manifests.json"
	}

	// 2. Fetch the manifest
	resp, err := http.Get(manifestURL)
	if err != nil {
		RespondError(w, "Failed to reach manifest registry", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		RespondError(w, "Manifest registry returned an error", http.StatusBadGateway)
		return
	}

	// 3. Proxy the JSON back to the Vue frontend
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, resp.Body)
}
