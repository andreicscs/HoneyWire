package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/honeywire/hub/internal/models"
	composesvc "github.com/honeywire/hub/internal/services/compose"
	"github.com/honeywire/hub/internal/services/sensor"
)

type SensorHandler struct {
	service        *sensor.Service
	composeService *composesvc.Service
}

func NewSensorHandler(svc *sensor.Service, composeSvc *composesvc.Service) *SensorHandler {
	return &SensorHandler{service: svc, composeService: composeSvc}
}

func (h *SensorHandler) ReceiveHeartbeat(w http.ResponseWriter, r *http.Request) {
	var hb models.Heartbeat
	if err := json.NewDecoder(r.Body).Decode(&hb); err != nil {
		RespondError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if hb.SensorID == "" {
		RespondError(w, "sensorId is required", http.StatusBadRequest)
		return
	}

	nodeID := NodeIDFromContext(r.Context())
	if nodeID == "" {
		RespondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Hand off to the Service layer
	if err := h.service.ProcessHeartbeat(nodeID, hb.SensorID, hb.Metadata); err != nil {
		if errors.Is(err, sensor.ErrSensorNotRegistered) {
			RespondError(w, "Sensor not registered on this node", http.StatusNotFound)
			return
		}
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

func (h *SensorHandler) ReceiveOffline(w http.ResponseWriter, r *http.Request) {
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

	nodeID := NodeIDFromContext(r.Context())
	if nodeID == "" {
		RespondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.service.ProcessOffline(nodeID, req.SensorID, req.Reason); err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "offline_acknowledged"})
}

func (h *SensorHandler) ToggleSilence(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeId")
	sensorID := chi.URLParam(r, "sensorId")

	var req struct {
		IsSilenced bool `json:"isSilenced"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.service.ToggleSilence(nodeID, sensorID, req.IsSilenced); err != nil {
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
func (h *SensorHandler) GetManifests(w http.ResponseWriter, r *http.Request) {
	body, err := h.composeService.FetchManifestBytes()
	if err != nil {
		RespondError(w, "Failed to reach manifest registry", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// codeql[go/xss] Writing safe JSON/YAML API response.
	// nosemgrep: go.lang.security.audit.xss.no-direct-write-to-responsewriter.no-direct-write-to-responsewriter
	w.Write(body)
}
