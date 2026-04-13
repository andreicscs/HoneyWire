package api

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/honeywire/hub/internal/auth"
	"github.com/honeywire/hub/internal/config"
	"github.com/honeywire/hub/internal/models"
	"github.com/honeywire/hub/internal/notify"
	"github.com/honeywire/hub/internal/store"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// --- Brute-Force Protection State ---
type loginState struct {
	attempts    int
	lockedUntil time.Time
}

var (
	authTracker = make(map[string]*loginState)
	authMutex   sync.Mutex
)

type Handler struct {
	Store        *store.Store
	Cfg          *config.Config
	SessionStore *auth.SessionStore

	clients   map[*websocket.Conn]bool
	clientsMu sync.Mutex
}

func NewHandler(s *store.Store, cfg *config.Config, sess *auth.SessionStore) *Handler {
	h := &Handler{
		Store:        s,
		Cfg:          cfg,
		SessionStore: sess,
		clients:      make(map[*websocket.Conn]bool),
	}

	go h.cleanupAuthTracker()
	return h
}

// Background routine to prevent memory leaks from abandoned IPs
func (h *Handler) cleanupAuthTracker() {
	for {
		time.Sleep(5 * time.Minute)
		authMutex.Lock()
		now := time.Now()
		for ip, state := range authTracker {
			if now.After(state.lockedUntil) {
				delete(authTracker, ip)
			}
		}
		authMutex.Unlock()
	}
}

// --- Helpers ---
func (h *Handler) broadcastWS(msgType string, payload interface{}) {
	var deadClients []*websocket.Conn
	h.clientsMu.Lock()
	for client := range h.clients {
		err := client.WriteJSON(map[string]interface{}{
			"type":    msgType,
			"payload": payload,
		})
		if err != nil {
			deadClients = append(deadClients, client)
		}
	}
	for _, c := range deadClients {
		delete(h.clients, c)
		c.Close()
	}
	h.clientsMu.Unlock()
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

func SendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("SendJSON encode error: %v\n", err)
	}
}

// --- WebSocket ---
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	h.clientsMu.Lock()
	h.clients[conn] = true
	h.clientsMu.Unlock()

	go func() {
		defer func() {
			h.clientsMu.Lock()
			delete(h.clients, conn)
			h.clientsMu.Unlock()
			conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

// --- Dashboard API ---
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
		numBlocks, delta, expectedPings = 30, 2*time.Minute, 2
	case "7D":
		numBlocks, delta, expectedPings = 7, 24*time.Hour, 1440
	case "30D":
		numBlocks, delta, expectedPings = 30, 24*time.Hour, 1440
	case "24H":
		fallthrough
	default:
		numBlocks, delta, expectedPings = 24, time.Hour, 60
	}

	cutoff := now.Add(-delta * time.Duration(numBlocks))
	cutoffStr := cutoff.Format("2006-01-02 15:04:05")

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
	history := make(map[string][]float64)

	for sensorRows.Next() {
		var s SensorData
		var lastSeenStr string
		sensorRows.Scan(&s.ID, &lastSeenStr, &s.FirstSeen)
		s.LastSeen, _ = time.Parse("2006-01-02 15:04:05", lastSeenStr)
		sensors = append(sensors, s)
		history[s.ID] = make([]float64, numBlocks)
	}

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

		idx := int(parsedBucket.Sub(cutoff) / delta)
		if idx >= numBlocks {
			idx = numBlocks - 1
		}

		if idx >= 0 && history[sID] != nil {
			history[sID][idx]++
		}
	}

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

			if blockEnd.Before(firstSeenParsed) {
				status, label = "nodata", "No Data (Not Deployed Yet)"
			} else {
				pings := history[s.ID][i]
				targetPings := expectedPings

				if firstSeenParsed.After(blockStart) && firstSeenParsed.Before(blockEnd) {
					activeDuration := blockEnd.Sub(firstSeenParsed)
					targetPings = activeDuration.Minutes()
					if targetPings > expectedPings {
						targetPings = expectedPings
					}
					if targetPings < 1 && activeDuration > 0 {
						targetPings = 1
					}
				} else if i == numBlocks-1 {
					activeDuration := now.Sub(blockStart)
					targetPings = activeDuration.Minutes()
					if targetPings > expectedPings {
						targetPings = expectedPings
					}
					if targetPings < 1 && activeDuration > 0 {
						targetPings = 1
					}
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

func (h *Handler) GetSystemState(w http.ResponseWriter, r *http.Request) {
	var isArmedStr string
	h.Store.DB.QueryRow("SELECT value FROM config WHERE key='is_armed'").Scan(&isArmedStr)
	SendJSON(w, http.StatusOK, map[string]bool{"is_armed": isArmedStr == "true"})
}

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

func (h *Handler) HandleVersion(w http.ResponseWriter, r *http.Request) {
	SendJSON(w, http.StatusOK, map[string]string{"version": h.Cfg.Version})
}

func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	var count int
	err := h.Store.DB.QueryRow("SELECT COUNT(*) FROM events WHERE is_read = 0 AND is_archived = 0").Scan(&count)
	if err != nil {
		count = 0
	}
	SendJSON(w, http.StatusOK, map[string]int{"count": count})
}

// --- Event Mutations ---
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

// DELETE /api/v1/events
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

	h.broadcastWS("SILENCE_SENSOR", map[string]interface{}{
		"sensor_id":   sensorID,
		"is_silenced": req.IsSilenced,
	})

	SendJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "success",
		"sensor_id":   sensorID,
		"is_silenced": req.IsSilenced,
	})
}

func (h *Handler) ForgetSensor(w http.ResponseWriter, r *http.Request) {
	sensorID := chi.URLParam(r, "sensor_id")
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

	h.broadcastWS("DELETE_SENSOR", map[string]string{"sensor_id": sensorID})

	SendJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Sensor forgotten successfully",
	})
}

// --- Agent Endpoints ---
func (h *Handler) ReceiveHeartbeat(w http.ResponseWriter, r *http.Request) {
	var hb models.Heartbeat
	if err := json.NewDecoder(r.Body).Decode(&hb); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	nowStr := now.Format("2006-01-02 15:04:05")
	minuteBucket := now.Format("2006-01-02 15:04:00")
	metadataJSON, _ := json.Marshal(hb.Metadata)

	// SQLite ON CONFLICT RowsAffected fix: 
	// Try insert-only first to reliably detect genuine new rows
	res, err := h.Store.DB.Exec(`
		INSERT OR IGNORE INTO sensors (sensor_id, first_seen, last_seen, metadata)
		VALUES (?, ?, ?, ?)`,
		hb.SensorID, nowStr, nowStr, string(metadataJSON),
	)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	affected, _ := res.RowsAffected()
	isNew := affected == 1

	// Update existing record
	h.Store.DB.Exec(`UPDATE sensors SET last_seen=?, metadata=? WHERE sensor_id=?`,
		nowStr, string(metadataJSON), hb.SensorID)

	// Log historical bucket
	h.Store.DB.Exec("INSERT OR IGNORE INTO sensor_heartbeats (sensor_id, time_bucket) VALUES (?, ?)", hb.SensorID, minuteBucket)

	if isNew {
		h.broadcastWS("NEW_SENSOR", map[string]string{"sensor_id": hb.SensorID})
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

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

	var isArmedStr string
	h.Store.DB.QueryRow("SELECT value FROM config WHERE key='is_armed'").Scan(&isArmedStr)

	var isSilencedInt int
	h.Store.DB.QueryRow("SELECT is_silenced FROM sensors WHERE sensor_id = ?", e.SensorID).Scan(&isSilencedInt)

	msg := "[" + e.SensorID + "] " + strings.ToUpper(e.EventTrigger) + " — " + e.Source + " -> " + e.Target

	if isArmedStr == "true" && isSilencedInt == 0 {
		notify.Dispatch(h.Cfg, "HoneyWire Alert ("+strings.ToUpper(e.Severity)+")", msg, e.Severity)
	}

	e.ID = int(lastInsertID)
	e.Timestamp = nowStr

	h.broadcastWS("NEW_EVENT", e)
	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// --- Auth ---
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ip := h.getRealIP(r)

	authMutex.Lock()
	if state, exists := authTracker[ip]; exists {
		// FIX: Only check the lockout timer if they actually hit the 10-attempt limit!
		if state.attempts >= 10 {
			if time.Now().Before(state.lockedUntil) {
				authMutex.Unlock()
				http.Error(w, "Too many failed attempts. Try again later.", http.StatusTooManyRequests)
				return
			}
			// Lockout expired, wipe the slate clean
			delete(authTracker, ip)
		}
	}
	authMutex.Unlock()

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if subtle.ConstantTimeCompare([]byte(req.Password), []byte(h.Cfg.DashboardPassword)) == 1 {
		authMutex.Lock()
		delete(authTracker, ip)
		authMutex.Unlock()

		token, err := h.SessionStore.Create()
		if err != nil {
			http.Error(w, "Session creation failed", http.StatusInternalServerError)
			return
		}

		isProd := h.Cfg.Env == "production"
		http.SetCookie(w, &http.Cookie{
			Name:     auth.CookieName,
			Value:    token,
			MaxAge:   2592000,
			HttpOnly: true,
			Secure:   isProd,
			SameSite: http.SameSiteStrictMode,
			Path:     "/",
		})

		SendJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	authMutex.Lock()
	if _, exists := authTracker[ip]; !exists {
		authTracker[ip] = &loginState{}
	}
	authTracker[ip].attempts++
	
	if authTracker[ip].attempts >= 10 {
		authTracker[ip].lockedUntil = time.Now().Add(15 * time.Minute)
		log.Printf("[!] AUDIT: IP %s locked out of dashboard for 15 minutes due to brute-force", ip)
	}
	authMutex.Unlock()

	http.Error(w, "Invalid Password", http.StatusUnauthorized)
}

// POST /logout
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