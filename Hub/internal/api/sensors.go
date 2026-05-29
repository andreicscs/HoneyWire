package api

import (
	"encoding/json"
	"net/http"

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

	if hb.SensorID == "" {
		RespondError(w, "sensorId is required", http.StatusBadRequest)
		return
	}

	nodeID, err := h.authenticateNodeRequest(r)
	if err != nil {
		RespondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Hand off to the Service layer
	if err := h.SensorService.ProcessHeartbeat(nodeID, hb.SensorID, hb.Metadata); err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

func (h *Handler) ReceiveOffline(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SensorID string `json:"sensorId"`
		Reason   string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.SensorID == "" {
		RespondError(w, "sensorId is required", http.StatusBadRequest)
		return
	}

	nodeID, err := h.authenticateNodeRequest(r)
	if err != nil {
		RespondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.SensorService.ProcessOffline(nodeID, req.SensorID, req.Reason); err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "offline_acknowledged"})
}

func (h *Handler) ToggleSilence(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeId")
	sensorID := chi.URLParam(r, "sensorId")

	var req struct {
		IsSilenced bool `json:"isSilenced"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.SensorService.ToggleSilence(nodeID, sensorID, req.IsSilenced); err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "success",
		"nodeId":     nodeID,
		"sensorId":   sensorID,
		"isSilenced": req.IsSilenced,
	})
}

// GetManifests fetches the sensor manifest JSON.
// Note: Authentication strategies generally stay in the HTTP layer since they interact directly with headers/cookies.
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

	// Assuming fetchManifestBytes is defined elsewhere in the api package
	body, err := fetchManifestBytes()
	if err != nil {
		RespondError(w, "Failed to reach manifest registry", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}