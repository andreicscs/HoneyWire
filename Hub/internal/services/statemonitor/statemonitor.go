package statemonitor

import (
	"context"
	"database/sql"
	"log"
	"time"
)

type Broadcaster interface {
	Broadcast(topic string, payload interface{})
}

type StateMonitor struct {
	db          *sql.DB
	broadcaster Broadcaster
}

func NewStateMonitor(db *sql.DB, broadcaster Broadcaster) *StateMonitor {
	return &StateMonitor{
		db:          db,
		broadcaster: broadcaster,
	}
}

func (s *StateMonitor) Start(ctx context.Context) {
	log.Println("[StateMonitor] Worker started.")
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[StateMonitor] StateMonitor stopped.")
			return
		case <-ticker.C:
			s.monitor()
		}
	}
}

func (s *StateMonitor) monitor() {
	cutoff := time.Now().UTC().Add(-60 * time.Second).Format(time.RFC3339)

	// Query for sensors where last_heartbeat is older than 60 seconds (i.e. < cutoff)
	rows, err := s.db.Query(`
		SELECT node_id, sensor_id
		FROM node_sensors
		WHERE last_heartbeat < ? OR last_heartbeat IS NULL
	`, cutoff)
	if err != nil {
		log.Printf("[ERROR] StateMonitor failed to query node_sensors: %v", err)
		return
	}
	defer rows.Close()

	nowStr := time.Now().UTC().Format(time.RFC3339)

	for rows.Next() {
		var nodeID, sensorID string
		if err := rows.Scan(&nodeID, &sensorID); err != nil {
			continue
		}

		// Get the latest status
		var lastStatus sql.NullString
		err := s.db.QueryRow(`
			SELECT status 
			FROM sensor_status_changes 
			WHERE node_id = ? AND sensor_id = ? 
			ORDER BY timestamp DESC, id DESC 
			LIMIT 1
		`, nodeID, sensorID).Scan(&lastStatus)

		if err != nil && err != sql.ErrNoRows {
			continue
		}

		if lastStatus.Valid && lastStatus.String == "offline" {
			continue // Already offline
		}

		// It's not offline (either "online" or no status yet). Insert offline.
		_, err = s.db.Exec(`
			INSERT INTO sensor_status_changes (node_id, sensor_id, status, timestamp)
			VALUES (?, ?, 'offline', ?)
		`, nodeID, sensorID, nowStr)
		if err != nil {
			log.Printf("[WARNING] StateMonitor failed to insert offline status: %v", err)
			continue
		}

		s.broadcaster.Broadcast("UPDATE_NODE", map[string]interface{}{
			"id":              nodeID,
			"trigger_refresh": true,
		})
		s.broadcaster.Broadcast("SYNC_CHARTS", nil)
	}
}
