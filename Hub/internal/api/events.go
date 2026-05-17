package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/honeywire/hub/internal/models"
	"github.com/honeywire/hub/internal/notify"
	"github.com/honeywire/hub/internal/siem"
)

func (h *Handler) ReceiveEvent(w http.ResponseWriter, r *http.Request) {
	var e models.Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		RespondError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if e.NodeID == "" || e.SensorID == "" {
		RespondError(w, "node_id and sensor_id are required", http.StatusBadRequest)
		return
	}

	// Per-node authentication (API Key -> NodeID)
	if !h.validateNodeAuth(r, e.NodeID) {
		RespondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Version check
	hubMajor := strings.Split(h.Cfg.Version, ".")[0]
	agentMajor := strings.Split(e.ContractVersion, ".")[0]
	if agentMajor == "" || hubMajor != agentMajor {
		RespondError(w, "Upgrade Required", http.StatusUpgradeRequired)
		return
	}

	nowStr := time.Now().UTC().Format(time.RFC3339)
	detailsJSON, _ := json.Marshal(e.Details)

	// Insert event with composite key reference (node_id, sensor_id)
	lastInsertID, err := h.Store.InsertEvent(&e, nowStr, string(detailsJSON))
	if err != nil {
		log.Printf("[ERROR] Failed to insert event for node %s/sensor %s: %v", e.NodeID, e.SensorID, err)
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}

	e.ID = lastInsertID
	e.Timestamp = nowStr

	// update node and sensor last_heartbeat (an event proves it is alive)
	h.Store.UpdateNodeLastHeartbeat(e.NodeID, e.SensorID, nowStr)

	// Check if sensor is silenced
	isSilenced, err := h.Store.IsSensorSilenced(e.NodeID, e.SensorID)
	if err != nil {
		log.Printf("[WARNING] Failed to check silence status for node %s/sensor %s: %v", e.NodeID, e.SensorID, err)
	}

	if !isSilenced {
		title := fmt.Sprintf("Intrusion Alert: %s", e.SensorID)
		message := fmt.Sprintf("Trigger: %s\nSource: %s\nTarget: %s", e.EventTrigger, e.Source, e.Target)
		notify.Dispatch(title, message, e.Severity)
	}

	select {
	case siem.EventQueue <- e:
		// Successfully queued for SIEM forwarder
	default:
		log.Println("[!] SIEM Queue full, dropping event")
	}

	h.broadcastWS("NEW_EVENT", e)
	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *Handler) GetEvents(w http.ResponseWriter, r *http.Request) {
	archivedParam := r.URL.Query().Get("archived")
	isArchived := 0
	if archivedParam == "true" {
		isArchived = 1
	}

	nodeID := r.URL.Query().Get("node_id")
	sensorID := r.URL.Query().Get("sensor_id")

	events, err := h.Store.GetEvents(isArchived, nodeID, sensorID)
	if err != nil {
		RespondError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, events)
}

func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	count, err := h.Store.GetUnreadEventCount()
	if err != nil {
		count = 0
	}
	SendJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (h *Handler) MarkSingleEventRead(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "event_id")
	if err := h.Store.MarkEventRead(eventID); err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}
	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *Handler) MarkEventsRead(w http.ResponseWriter, r *http.Request) {
	if err := h.Store.MarkAllEventsRead(); err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}
	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *Handler) ArchiveEvent(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "event_id")
	if err := h.Store.ArchiveEvent(eventID); err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}
	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *Handler) ArchiveAll(w http.ResponseWriter, r *http.Request) {
	if err := h.Store.ArchiveAllEvents(); err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}
	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *Handler) ClearEvents(w http.ResponseWriter, r *http.Request) {
	dryrun := r.URL.Query().Get("dryrun") == "true"

	if dryrun {
		count, _ := h.Store.GetEventCount()
		SendJSON(w, http.StatusOK, map[string]interface{}{
			"status":       "success",
			"dryrun":       true,
			"would_delete": count,
		})
		return
	}

	ip := h.getRealIP(r)
	log.Printf("[!] AUDIT: Database purged by IP %s", ip)

	if err := h.Store.ClearAllEvents(); err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	SendJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"dryrun": false,
	})
}