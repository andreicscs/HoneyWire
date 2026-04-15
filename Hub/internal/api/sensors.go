package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/honeywire/hub/internal/models"
)

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

	h.Store.DB.Exec(`UPDATE sensors SET last_seen=?, metadata=? WHERE sensor_id=?`,
		nowStr, string(metadataJSON), hb.SensorID)

	h.Store.DB.Exec("INSERT OR IGNORE INTO sensor_heartbeats (sensor_id, time_bucket) VALUES (?, ?)", hb.SensorID, minuteBucket)

	if isNew {
		h.broadcastWS("NEW_SENSOR", map[string]string{"sensor_id": hb.SensorID})
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

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