package store

import (
	"encoding/json"
	"time"

	"github.com/honeywire/hub/internal/models"
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

func (s *SQLiteStore) UpsertSensor(hb *models.Heartbeat, nowStr, metadataStr string) error {
	_, err := s.DB.Exec(`
		INSERT INTO sensors (node_id, sensor_id, first_seen, last_seen, metadata, is_silenced)
		VALUES (?, ?, ?, ?, ?, 0)
		ON CONFLICT(node_id, sensor_id) DO UPDATE SET last_seen = ?, metadata = ?`,
		hb.NodeID, hb.SensorID, nowStr, nowStr, metadataStr, nowStr, metadataStr,
	)
	return err
}

func (s *SQLiteStore) InsertHeartbeat(nodeID, sensorID, timeBucket string) error {
	_, err := s.DB.Exec(
		"INSERT OR IGNORE INTO sensor_heartbeats (node_id, sensor_id, time_bucket) VALUES (?, ?, ?)",
		nodeID, sensorID, timeBucket,
	)
	return err
}

func (s *SQLiteStore) GetAllSensors() ([]models.Sensor, error) {
	rows, err := s.DB.Query(`
		SELECT sensor_id, node_id, first_seen, last_seen, metadata, is_silenced 
		FROM sensors 
		ORDER BY COALESCE(node_id, 'ZZZ') ASC, sensor_id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fleet []models.Sensor
	for rows.Next() {
		var s models.Sensor
		var metadataStr string
		var isSilencedInt int
		var dbNodeID *string

		if err := rows.Scan(&s.SensorID, &dbNodeID, &s.FirstSeen, &s.LastSeen, &metadataStr, &isSilencedInt); err != nil {
			continue
		}

		if dbNodeID != nil {
			s.NodeID = *dbNodeID
		}

		s.IsSilenced = isSilencedInt == 1

		lastSeenTime, err := time.Parse(time.RFC3339, s.LastSeen)
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
	return fleet, nil
}

// GetSensor retrieves a single sensor by node_id and sensor_id composite key
func (s *SQLiteStore) GetSensor(nodeID, sensorID string) (*models.Sensor, error) {
	var sensor models.Sensor
	var metadataStr string
	var isSilencedInt int

	err := s.DB.QueryRow(
		`SELECT sensor_id, node_id, first_seen, last_seen, metadata, is_silenced 
		 FROM sensors 
		 WHERE node_id = ? AND sensor_id = ?`,
		nodeID, sensorID,
	).Scan(&sensor.SensorID, &sensor.NodeID, &sensor.FirstSeen, &sensor.LastSeen, &metadataStr, &isSilencedInt)

	if err != nil {
		return nil, err
	}

	sensor.IsSilenced = isSilencedInt == 1

	lastSeenTime, err := time.Parse(time.RFC3339, sensor.LastSeen)
	if err == nil && time.Now().UTC().Sub(lastSeenTime) < 90*time.Second {
		sensor.Status = "online"
	} else {
		sensor.Status = "offline"
	}

	var metadata map[string]interface{}
	json.Unmarshal([]byte(metadataStr), &metadata)
	sensor.Metadata = metadata

	return &sensor, nil
}

func (s *SQLiteStore) GetSensorsForUptime(nowStr string) ([]SensorUptimeData, error) {
	rows, err := s.DB.Query("SELECT node_id, sensor_id, last_seen, COALESCE(first_seen, ?) FROM sensors ORDER BY node_id, sensor_id", nowStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sensors []SensorUptimeData
	for rows.Next() {
		var sd SensorUptimeData
		var lastSeenStr string
		if err := rows.Scan(&sd.NodeID, &sd.SensorID, &lastSeenStr, &sd.FirstSeen); err == nil {
			sd.LastSeen, _ = time.Parse(time.RFC3339, lastSeenStr)
			sensors = append(sensors, sd)
		}
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
		if err := rows.Scan(&hb.NodeID, &hb.SensorID, &hb.TimeBucket); err == nil {
			hbs = append(hbs, hb)
		}
	}
	return hbs, nil
}

func (s *SQLiteStore) UpdateSensorSilence(nodeID, sensorID string, silenceVal int) error {
	_, err := s.DB.Exec("UPDATE sensors SET is_silenced = ? WHERE node_id = ? AND sensor_id = ?", silenceVal, nodeID, sensorID)
	return err
}

func (s *SQLiteStore) DeleteSensor(nodeID, sensorID string) (int64, error) {
	result, err := s.DB.Exec("DELETE FROM sensors WHERE node_id = ? AND sensor_id = ?", nodeID, sensorID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
