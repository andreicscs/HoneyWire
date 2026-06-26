package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/honeywire/hub/internal/models"
	"github.com/honeywire/hub/internal/services/config"
	"github.com/honeywire/hub/internal/services/event"
)

type EventHandler struct {
	service *event.Service
	Cfg     *config.Config
}

func NewEventHandler(svc *event.Service, cfg *config.Config) *EventHandler {
	return &EventHandler{service: svc, Cfg: cfg}
}

func (h *EventHandler) ReceiveEvent(w http.ResponseWriter, r *http.Request) {
	var e models.Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		RespondError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if e.SensorID == "" {
		RespondError(w, "sensorId is required", http.StatusBadRequest)
		return
	}

	nodeID := NodeIDFromContext(r.Context())

	if nodeID == "" {
		RespondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.service.ProcessEvent(&e, nodeID); err != nil {
		if err.Error() == "upgrade required" {
			RespondError(w, "Upgrade Required", http.StatusUpgradeRequired)
			return
		}
		if err.Error() == "sensor_not_registered" {
			RespondError(w, "Sensor is not registered", http.StatusNotFound)
			return
		}
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *EventHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	archivedParam := r.URL.Query().Get("archived")
	isArchived := 0
	if archivedParam == "true" {
		isArchived = 1
	}

	nodeID := r.URL.Query().Get("nodeId")
	sensorID := r.URL.Query().Get("sensorId")

	events, err := h.service.GetEvents(isArchived, nodeID, sensorID)
	if err != nil {
		RespondError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, events)
}

func (h *EventHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	count, err := h.service.GetUnreadCount()
	if err != nil {
		count = 0
	}
	SendJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (h *EventHandler) MarkSingleEventRead(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventId")
	if err := h.service.MarkSingleEventRead(eventID); err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}
	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *EventHandler) MarkEventsRead(w http.ResponseWriter, r *http.Request) {
	if err := h.service.MarkEventsRead(); err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}
	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *EventHandler) ExportEvents(w http.ResponseWriter, r *http.Request) {
	archivedParam := r.URL.Query().Get("archived")
	isArchived := 0
	if archivedParam == "true" {
		isArchived = 1
	}

	nodeID := r.URL.Query().Get("nodeId")
	sensorID := r.URL.Query().Get("sensorId")

	events, err := h.service.GetEvents(isArchived, nodeID, sensorID)
	if err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=honeywire_events.json")
	w.Header().Set("Content-Type", "application/json")
	
	if err := json.NewEncoder(w).Encode(events); err != nil {
		http.Error(w, "Failed to encode events", http.StatusInternalServerError)
		return
	}
}

func (h *EventHandler) ArchiveEvent(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventId")
	if err := h.service.ArchiveEvent(eventID); err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}
	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *EventHandler) ArchiveAll(w http.ResponseWriter, r *http.Request) {
	if err := h.service.ArchiveAll(); err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}
	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *EventHandler) ClearEvents(w http.ResponseWriter, r *http.Request) {
	dryrun := r.URL.Query().Get("dryrun") == "true"
	ip := GetRealIP(r, h.Cfg.TrustProxy)

	count, err := h.service.ClearEvents(dryrun, ip)

	if dryrun {
		SendJSON(w, http.StatusOK, map[string]interface{}{
			"status":       "success",
			"dryrun":       true,
			"would_delete": count,
		})
		return
	}

	if err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"dryrun": false,
	})
}
