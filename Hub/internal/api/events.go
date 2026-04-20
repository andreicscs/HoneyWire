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
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	hubMajor := strings.Split(h.Cfg.Version, ".")[0]
	agentMajor := strings.Split(e.ContractVersion, ".")[0]
	if agentMajor == "" || hubMajor != agentMajor {
		http.Error(w, "Upgrade Required", http.StatusUpgradeRequired)
		return
	}

	nowStr := time.Now().UTC().Format("2006-01-02 15:04:05")
	detailsJSON, _ := json.Marshal(e.Details)

	result, err := h.Store.DB.Exec(`
		INSERT INTO events (timestamp, contract_version, sensor_id, event_trigger, severity, source, target, details, is_read, is_archived)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, 0)`,
		nowStr, e.ContractVersion, e.SensorID, e.EventTrigger, e.Severity, e.Source, e.Target, string(detailsJSON),
	)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	lastInsertID, _ := result.LastInsertId()
	e.ID = int(lastInsertID)
	e.Timestamp = nowStr

	var isSilencedInt int
	h.Store.DB.QueryRow("SELECT is_silenced FROM sensors WHERE sensor_id = ?", e.SensorID).Scan(&isSilencedInt)

	if isSilencedInt == 0 {
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

	query := "SELECT id, timestamp, contract_version, sensor_id, event_trigger, severity, source, target, details, is_read, is_archived FROM events WHERE is_archived = ?"
	args := []interface{}{isArchived}

	if sensorID := r.URL.Query().Get("sensor_id"); sensorID != "" {
		query += " AND sensor_id = ?"
		args = append(args, sensorID)
	}

	query += " ORDER BY id DESC"

	rows, err := h.Store.DB.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var e models.Event
		var detailsStr string
		var isReadInt, isArchivedInt int

		if err := rows.Scan(
			&e.ID, &e.Timestamp, &e.ContractVersion, &e.SensorID,
			&e.EventTrigger, &e.Severity, &e.Source, &e.Target,
			&detailsStr, &isReadInt, &isArchivedInt,
		); err != nil {
			continue
		}

		e.IsRead = isReadInt == 1
		e.IsArchived = isArchivedInt == 1
		json.Unmarshal([]byte(detailsStr), &e.Details)
		events = append(events, e)
	}

	if events == nil {
		events = []models.Event{}
	}

	SendJSON(w, http.StatusOK, events)
}

func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	var count int
	err := h.Store.DB.QueryRow("SELECT COUNT(*) FROM events WHERE is_read = 0 AND is_archived = 0").Scan(&count)
	if err != nil {
		count = 0
	}
	SendJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (h *Handler) MarkSingleEventRead(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "event_id")
	h.Store.DB.Exec("UPDATE events SET is_read = 1 WHERE id = ?", eventID)
	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *Handler) MarkEventsRead(w http.ResponseWriter, r *http.Request) {
	h.Store.DB.Exec("UPDATE events SET is_read = 1 WHERE is_read = 0")
	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *Handler) ArchiveEvent(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "event_id")
	h.Store.DB.Exec("UPDATE events SET is_archived = 1, is_read = 1 WHERE id = ?", eventID)
	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *Handler) ArchiveAll(w http.ResponseWriter, r *http.Request) {
	h.Store.DB.Exec("UPDATE events SET is_archived = 1, is_read = 1 WHERE is_archived = 0")
	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *Handler) ClearEvents(w http.ResponseWriter, r *http.Request) {
	dryrun := r.URL.Query().Get("dryrun") == "true"

	if dryrun {
		var count int
		h.Store.DB.QueryRow("SELECT COUNT(*) FROM events").Scan(&count)
		SendJSON(w, http.StatusOK, map[string]interface{}{
			"status":       "success",
			"dryrun":       true,
			"would_delete": count,
		})
		return
	}

	ip := h.getRealIP(r)
	log.Printf("[!] AUDIT: Database purged by IP %s", ip)

	h.Store.DB.Exec("DELETE FROM events")
	SendJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"dryrun": false,
	})
}
