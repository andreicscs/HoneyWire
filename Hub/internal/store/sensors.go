package store

import (
	"database/sql"
	"encoding/json"
	"time"
)

type SensorUptimeData struct {
	NodeID    string
	SensorID  string
	LastSeen  time.Time
	FirstSeen string
}

type HeartbeatData struct {
	NodeID     string
	SensorID   string
	TimeBucket string
}

// ProcessHeartbeat safely handles node updates and config reconciliation
func (s *SQLiteStore) ProcessHeartbeat(nodeID, sensorID, agentRevision, nowStr, metadataStr string) (bool, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback() // Safely catches any uncommitted exit

	// Update the Host Node's heartbeat
	if _, err := tx.Exec("UPDATE nodes SET last_heartbeat = ? WHERE id = ?", nowStr, nodeID); err != nil {
		return false, err
	}

	// Update the specific Sensor's heartbeat AND metadata
	if _, err := tx.Exec(`
		UPDATE node_sensors 
		SET metadata = ?, last_heartbeat = ?, updated_at = ? 
		WHERE node_id = ? AND sensor_id = ?`,
		metadataStr, nowStr, nowStr, nodeID, sensorID); err != nil {
		return false, err
	}

	// Check config reconciliation
	var activeRevision sql.NullString
	var desiredRevision sql.NullString
	var pendingConfig int
	err = tx.QueryRow("SELECT active_revision, desired_revision, pending_config FROM nodes WHERE id = ?", nodeID).Scan(&activeRevision, &desiredRevision, &pendingConfig)

	justSynced := false
	if err == nil && pendingConfig == 1 {
		targetRevision := ""
		if desiredRevision.Valid {
			targetRevision = desiredRevision.String
		}

		if targetRevision != "" && targetRevision == agentRevision {
			rows, err := tx.Query("SELECT metadata FROM node_sensors WHERE node_id = ?", nodeID)
			if err == nil {
				allMatched := true
				for rows.Next() {
					var metaStr string
					if err := rows.Scan(&metaStr); err != nil {
						allMatched = false
						break
					}
					var meta map[string]interface{}
					if err := json.Unmarshal([]byte(metaStr), &meta); err != nil {
						allMatched = false
						break
					}
					if rev, ok := meta["HW_CONFIG_REV"].(string); !ok || rev != targetRevision {
						allMatched = false
						break
					}
				}
				rows.Close()

				if allMatched {
					if _, err := tx.Exec(`UPDATE nodes SET active_revision = ?, desired_revision = NULL, pending_config = 0 WHERE id = ?`, targetRevision, nodeID); err != nil {
						return false, err
					}
					justSynced = true
				}
			}
		}
	}

	return justSynced, tx.Commit()
}

func (s *SQLiteStore) InsertHeartbeat(nodeID, sensorID, timeBucket string) error {
	_, err := s.DB.Exec(
		"INSERT OR IGNORE INTO sensor_heartbeats (node_id, sensor_id, time_bucket) VALUES (?, ?, ?)",
		nodeID, sensorID, timeBucket,
	)
	return err
}

func (s *SQLiteStore) GetSensorsForUptime(nowStr string) ([]SensorUptimeData, error) {
	// Join nodes and node_sensors to get the creation date and the parent node's last heartbeat
	rows, err := s.DB.Query(`
		SELECT ns.node_id, ns.sensor_id, n.last_heartbeat, ns.created_at 
		FROM node_sensors ns
		JOIN nodes n ON ns.node_id = n.id
		ORDER BY ns.node_id, ns.sensor_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sensors []SensorUptimeData
	for rows.Next() {
		var sd SensorUptimeData
		var lastSeenStr sql.NullString

		if err := rows.Scan(&sd.NodeID, &sd.SensorID, &lastSeenStr, &sd.FirstSeen); err != nil {
			return nil, err
		}

		if lastSeenStr.Valid {
			sd.LastSeen, _ = time.Parse(time.RFC3339, lastSeenStr.String)
		} else {
			sd.LastSeen = time.Time{} // Never checked in
		}
		sensors = append(sensors, sd)
	}
	return sensors, nil
}

func (s *SQLiteStore) GetHeartbeatsSince(cutoffStr string) ([]HeartbeatData, error) {
	rows, err := s.DB.Query("SELECT node_id, sensor_id, time_bucket FROM sensor_heartbeats WHERE time_bucket >= ?", cutoffStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hbs []HeartbeatData
	for rows.Next() {
		var hb HeartbeatData
		if err := rows.Scan(&hb.NodeID, &hb.SensorID, &hb.TimeBucket); err != nil {
			return nil, err
		}
		hbs = append(hbs, hb)
	}
	return hbs, nil
}

func (s *SQLiteStore) UpdateSensorSilence(nodeID, sensorID string, silenceVal int) error {
	_, err := s.DB.Exec("UPDATE node_sensors SET is_silenced = ? WHERE node_id = ? AND sensor_id = ?", silenceVal, nodeID, sensorID)
	return err
}

// UpdateNodeLastHeartbeat flags BOTH the parent node and the specific sensor as alive
func (s *SQLiteStore) UpdateNodeLastHeartbeat(nodeID, sensorID, timestamp string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`UPDATE nodes SET last_heartbeat = ? WHERE id = ?`, timestamp, nodeID); err != nil {
		tx.Rollback()
		return err
	}

	if _, err := tx.Exec(`UPDATE node_sensors SET last_heartbeat = ?, updated_at = ? WHERE node_id = ? AND sensor_id = ?`, timestamp, timestamp, nodeID, sensorID); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
