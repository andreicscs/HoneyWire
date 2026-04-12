package api

import (
	"encoding/json"
	"net/http"
	"time"
	"strings"
	"fmt"
	"crypto/subtle"

	"github.com/go-chi/chi/v5"
	"github.com/honeywire/hub/internal/notify"
	"github.com/honeywire/hub/internal/config"
	"github.com/honeywire/hub/internal/models"
	"github.com/honeywire/hub/internal/store"
	"github.com/honeywire/hub/internal/auth"
)

// Handler holds our dependencies so our API endpoints can access the database
type Handler struct {
	Store *store.Store
	Cfg   *config.Config
	SessionStore *auth.SessionStore
}

func NewHandler(s *store.Store, cfg *config.Config, sess *auth.SessionStore) *Handler {
	return &Handler{Store: s, Cfg: cfg, SessionStore: sess}
}

// Helper
func SendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// GET /api/v1/sensors
func (h *Handler) GetSensors(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Store.DB.Query("SELECT sensor_id, last_seen, metadata, is_silenced FROM sensors ORDER BY sensor_id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var fleet []models.Sensor
	for rows.Next() {
		var s models.Sensor
		var metadataStr string
		var isSilencedInt int

		if err := rows.Scan(&s.SensorID, &s.LastSeen, &metadataStr, &isSilencedInt); err != nil {
            continue 
        }

		s.IsSilenced = isSilencedInt == 1
		
		// "2006-01-02 15:04:05" is Go's magic reference date used to define YYYY-MM-DD HH:MM:SS
		lastSeenTime, err := time.Parse("2006-01-02 15:04:05", s.LastSeen)
		if err == nil && time.Now().UTC().Sub(lastSeenTime) < 90*time.Second {
			s.Status = "online"
		} else {
			s.Status = "offline"
		}

		var metadata map[string]interface{}
		json.Unmarshal([]byte(metadataStr), &metadata)
		s.Metadata = metadata

		fleet = append(fleet, s)
	}

	if fleet == nil {
		fleet = []models.Sensor{}
	}

	SendJSON(w, http.StatusOK, fleet)
}

// GET /api/v1/events
func (h *Handler) GetEvents(w http.ResponseWriter, r *http.Request) {
	archivedParam := r.URL.Query().Get("archived")
	isArchived := 0
	if archivedParam == "true" {
		isArchived = 1
	}

	rows, err := h.Store.DB.Query("SELECT id, timestamp, contract_version, sensor_id, event_trigger, severity, source, target, details, is_read, is_archived FROM events WHERE is_archived = ? ORDER BY id DESC", isArchived)
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
            &e.ID, &e.ContractVersion, &e.SensorID,
            &e.EventTrigger, &e.Severity, &e.Source, &e.Target,
            &detailsStr, &isReadInt, &isArchivedInt,
        ); err != nil {
            continue
        }

		e.IsRead = isReadInt == 1
		e.IsArchived = isArchivedInt == 1

		var details map[string]interface{}
		json.Unmarshal([]byte(detailsStr), &details)
		e.Details = details

		events = append(events, e)
	}

	if events == nil {
		events = []models.Event{}
	}

	SendJSON(w, http.StatusOK, events)
}

// POST /api/v1/heartbeat
func (h *Handler) ReceiveHeartbeat(w http.ResponseWriter, r *http.Request) {
	var hb models.Heartbeat
	if err := json.NewDecoder(r.Body).Decode(&hb); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	nowStr := now.Format("2006-01-02 15:04:05")
	minuteBucket := now.Format("2006-01-02 15:04:00") // Rounds down to minute

	metadataJSON, _ := json.Marshal(hb.Metadata)

	// 1. Update live status & first_seen
	_, err := h.Store.DB.Exec(`
        INSERT INTO sensors (sensor_id, first_seen, last_seen, metadata)
        VALUES (?, ?, ?, ?)
        ON CONFLICT(sensor_id) DO UPDATE SET last_seen=?, metadata=?`,
        hb.SensorID, nowStr, nowStr, string(metadataJSON),
        nowStr, string(metadataJSON),
    )
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// 2. Log historical bucket
	h.Store.DB.Exec("INSERT OR IGNORE INTO sensor_heartbeats (sensor_id, time_bucket) VALUES (?, ?)", hb.SensorID, minuteBucket)

	SendJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

// POST /api/v1/event
func (h *Handler) ReceiveEvent(w http.ResponseWriter, r *http.Request) {
	var e models.Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Basic Version Check
	hubMajor := strings.Split(h.Cfg.Version, ".")[0] 
	agentMajor := strings.Split(e.ContractVersion, ".")[0]
	if hubMajor != agentMajor {
		http.Error(w, "Upgrade Required", http.StatusUpgradeRequired)
		return
	}

	nowStr := time.Now().UTC().Format("2006-01-02 15:04:05")
	detailsJSON, _ := json.Marshal(e.Details)

	// Insert Event
	_, err := h.Store.DB.Exec(`
        INSERT INTO events (timestamp, contract_version, sensor_id, event_trigger, severity, source, target, details, is_read, is_archived)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, 0)`,
        nowStr, e.ContractVersion, e.SensorID, e.EventTrigger, e.Severity, e.Source, e.Target, string(detailsJSON),
    )
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Check Armed & Silenced States
	var isArmedStr string
	h.Store.DB.QueryRow("SELECT value FROM config WHERE key='is_armed'").Scan(&isArmedStr)
	
	var isSilencedInt int
	h.Store.DB.QueryRow("SELECT is_silenced FROM sensors WHERE sensor_id = ?", e.SensorID).Scan(&isSilencedInt)

	msg := "[" + e.SensorID + "] " + strings.ToUpper(e.EventTrigger) + " — " + e.Source + " -> " + e.Target 

    if isArmedStr == "true" && isSilencedInt == 0 {
        notify.Dispatch(h.Cfg, "HoneyWire Alert ("+strings.ToUpper(e.Severity)+")", msg, e.Severity)
    }

	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// PATCH /api/v1/events/{event_id}/archive
func (h *Handler) ArchiveEvent(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "event_id")
	
	_, err := h.Store.DB.Exec("UPDATE events SET is_archived = 1, is_read = 1 WHERE id = ?", eventID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// PATCH /api/v1/events/archive-all
func (h *Handler) ArchiveAll(w http.ResponseWriter, r *http.Request) {
	_, err := h.Store.DB.Exec("UPDATE events SET is_archived = 1, is_read = 1 WHERE is_archived = 0")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// PATCH /api/v1/sensors/{sensor_id}/silence
func (h *Handler) ToggleSilence(w http.ResponseWriter, r *http.Request) {
	sensorID := chi.URLParam(r, "sensor_id")
	
	var req struct {
		IsSilenced bool `json:"is_silenced"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	silenceVal := 0
	if req.IsSilenced {
		silenceVal = 1
	}

	_, err := h.Store.DB.Exec("UPDATE sensors SET is_silenced = ? WHERE sensor_id = ?", silenceVal, sensorID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "success",
		"sensor_id":   sensorID,
		"is_silenced": req.IsSilenced,
	})
}

// GET /api/v1/uptime
func (h *Handler) GetUptime(w http.ResponseWriter, r *http.Request) {
	timeframe := r.URL.Query().Get("timeframe")
	if timeframe == "" {
		timeframe = "24H"
	}

	now := time.Now().UTC()
	var numBlocks int
	var delta time.Duration
	var expectedPings float64

	switch timeframe {
	case "1H":
		numBlocks, delta, expectedPings = 30, 2*time.Minute, 2 // 30 pills, 2 min each
	case "7D":
		numBlocks, delta, expectedPings = 7, 24*time.Hour, 1440
	case "30D":
		numBlocks, delta, expectedPings = 30, 24*time.Hour, 1440
	case "24H":
		fallthrough
	default: // 24H (24 blocks, each 1 hour)
		numBlocks, delta, expectedPings = 24, time.Hour, 60
	}

	cutoff := now.Add(-delta * time.Duration(numBlocks))
	cutoffStr := cutoff.Format("2006-01-02 15:04:05")

	// 1. Get all sensors
	sensorRows, err := h.Store.DB.Query("SELECT sensor_id, last_seen, COALESCE(first_seen, ?) FROM sensors ORDER BY sensor_id", now.Format("2006-01-02 15:04:05"))
	if err != nil {
		http.Error(w, "Database error fetching sensors", http.StatusInternalServerError)
		return
	}
	defer sensorRows.Close()

	type SensorData struct {
		ID        string
		LastSeen  time.Time
		FirstSeen string
	}
	var sensors []SensorData
	
	// history will map sensorID to a slice of pings matching the numBlocks
	history := make(map[string][]float64)

	for sensorRows.Next() {
		var s SensorData
		var lastSeenStr string
		sensorRows.Scan(&s.ID, &lastSeenStr, &s.FirstSeen)
		s.LastSeen, _ = time.Parse("2006-01-02 15:04:05", lastSeenStr)
		sensors = append(sensors, s)
		history[s.ID] = make([]float64, numBlocks)
	}

	// 2. Get heartbeats and map them to block indexes purely via time math
	hbRows, err := h.Store.DB.Query("SELECT sensor_id, time_bucket FROM sensor_heartbeats WHERE time_bucket >= ?", cutoffStr)
	if err != nil {
		http.Error(w, "Database error fetching heartbeats", http.StatusInternalServerError)
		return
	}
	defer hbRows.Close()
	
	for hbRows.Next() {
		var sID, tBucket string
		hbRows.Scan(&sID, &tBucket)
		parsedBucket, _ := time.Parse("2006-01-02 15:04:00", tBucket)

		if parsedBucket.Before(cutoff) {
			continue
		}

		// Figure out which block this heartbeat falls into (0 = oldest, numBlocks-1 = newest)
		idx := int(parsedBucket.Sub(cutoff) / delta)
		if idx >= numBlocks {
			idx = numBlocks - 1
		}
		
		if idx >= 0 && history[sID] != nil {
			history[sID][idx]++
		}
	}

	// 3. Build the heatmap blocks
	var result []map[string]interface{}
	for _, s := range sensors {
		firstSeenParsed, _ := time.Parse("2006-01-02 15:04:05", s.FirstSeen)
		var blocks []map[string]string

		for i := 0; i < numBlocks; i++ {
			blockStart := cutoff.Add(time.Duration(i) * delta)
			blockEnd := blockStart.Add(delta)

			stepsAgo := numBlocks - 1 - i
			timeLabel := "Current"
			if stepsAgo > 0 {
				switch timeframe {
				case "1H":
					timeLabel = fmt.Sprintf("%d mins ago", stepsAgo*int(delta.Minutes()))
				case "24H":
					timeLabel = fmt.Sprintf("%d hours ago", stepsAgo)
				case "7D", "30D":
					timeLabel = fmt.Sprintf("%d days ago", stepsAgo)
				default:
					timeLabel = fmt.Sprintf("%d ago", stepsAgo)
				}
			}

			status, label := "", ""

			// Math checks
			if blockEnd.Before(firstSeenParsed) {
				status, label = "nodata", "No Data (Not Deployed Yet)"
			} else {
				pings := history[s.ID][i]
				targetPings := expectedPings

				// Adjust expectations for the deployment boundary and the live block boundary
				if firstSeenParsed.After(blockStart) && firstSeenParsed.Before(blockEnd) {
					activeDuration := blockEnd.Sub(firstSeenParsed)
					targetPings = activeDuration.Minutes()
					if targetPings > expectedPings { targetPings = expectedPings }
					if targetPings < 1 && activeDuration > 0 { targetPings = 1 }
				} else if i == numBlocks-1 {
					activeDuration := now.Sub(blockStart)
					targetPings = activeDuration.Minutes()
					if targetPings > expectedPings { targetPings = expectedPings }
					if targetPings < 1 && activeDuration > 0 { targetPings = 1 }
				}

				if pings == 0 && targetPings >= 1 {
					status, label = "down", "Offline"
				} else if targetPings > 0 && pings < (targetPings*0.85) {
					status, label = "degraded", fmt.Sprintf("Degraded (%.0f/%.0f pings)", pings, targetPings)
				} else {
					status, label = "up", "Online"
				}
			}

			blocks = append(blocks, map[string]string{
				"status":    status,
				"timeLabel": timeLabel,
				"label":     label,
			})
		}

		// Override final block for live status
		isLive := now.Sub(s.LastSeen) < 90*time.Second
		if isLive {
			blocks[len(blocks)-1]["status"] = "up"
			blocks[len(blocks)-1]["label"] = "Online (Live)"
		} else {
			blocks[len(blocks)-1]["status"] = "down"
			blocks[len(blocks)-1]["label"] = "Offline (Live)"
		}

		result = append(result, map[string]interface{}{
			"id":       s.ID,
			"name":     s.ID,
			"isOnline": isLive,
			"blocks":   blocks,
		})
	}

	if result == nil {
		result = []map[string]interface{}{}
	}
	SendJSON(w, http.StatusOK, result)
}

func (h *Handler) HandleVersion(w http.ResponseWriter, r *http.Request) {
	SendJSON(w, http.StatusOK, map[string]string{"version": h.Cfg.Version})
}

// GET /api/v1/system/state
func (h *Handler) GetSystemState(w http.ResponseWriter, r *http.Request) {
	var isArmedStr string
	h.Store.DB.QueryRow("SELECT value FROM config WHERE key='is_armed'").Scan(&isArmedStr)
	SendJSON(w, http.StatusOK, map[string]bool{"is_armed": isArmedStr == "true"})
}

// PATCH /api/v1/system/state
func (h *Handler) SetSystemState(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IsArmed bool `json:"is_armed"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	val := "false"
	if req.IsArmed {
		val = "true"
	}
	h.Store.DB.Exec("UPDATE config SET value=? WHERE key='is_armed'", val)
	SendJSON(w, http.StatusOK, map[string]interface{}{"status": "success", "is_armed": req.IsArmed})
}

// PATCH /api/v1/events/{event_id}/read
func (h *Handler) MarkSingleEventRead(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "event_id")
	h.Store.DB.Exec("UPDATE events SET is_read = 1 WHERE id = ?", eventID)
	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// PATCH /api/v1/events/read
func (h *Handler) MarkEventsRead(w http.ResponseWriter, r *http.Request) {
	h.Store.DB.Exec("UPDATE events SET is_read = 1 WHERE is_read = 0")
	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// DELETE /api/v1/events
func (h *Handler) ClearEvents(w http.ResponseWriter, r *http.Request) {
	h.Store.DB.Exec("DELETE FROM events")
	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// DELETE /api/v1/sensors/{sensor_id}
func (h *Handler) ForgetSensor(w http.ResponseWriter, r *http.Request) {
    sensorID := chi.URLParam(r, "sensor_id")
    
    // Explicitly delete the sensor. Historical events remain intact for auditing,
    // but the sensor configuration and heatbeats will be wiped from active monitoring.
    h.Store.DB.Exec("DELETE FROM sensor_heartbeats WHERE sensor_id = ?", sensorID)
    
    result, err := h.Store.DB.Exec("DELETE FROM sensors WHERE sensor_id = ?", sensorID)
    if err != nil {
        http.Error(w, "Database error while deleting sensor", http.StatusInternalServerError)
        return
    }

    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        http.Error(w, "Sensor not found", http.StatusNotFound)
        return
    }

    SendJSON(w, http.StatusOK, map[string]string{
        "status":  "success",
        "message": "Sensor forgotten successfully",
    })
}

// --- Auth & UI Handlers ---

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Use ConstantTimeCompare to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(req.Password), []byte(h.Cfg.DashboardPassword)) == 1 {
		token, err := h.SessionStore.Create()
		if err != nil {
			http.Error(w, "Session creation failed", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     auth.CookieName,
			Value:    token,
			MaxAge:   2592000, // 30 days
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Path:     "/",
		})

		SendJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	http.Error(w, "Invalid Password", http.StatusUnauthorized)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(auth.CookieName); err == nil {
		h.SessionStore.Delete(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		Path:     "/",
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
