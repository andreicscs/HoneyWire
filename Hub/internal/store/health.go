package store

import (
	"time"
)

// GetTransitionedOfflineNodes checks all nodes and sensors to see if their
// last_heartbeat crossed the offline threshold since the last check.
// It returns a set of Node IDs that transitioned to 'down' so the Hub can broadcast them.
func (s *SQLiteStore) GetTransitionedOfflineNodes(offlineThreshold time.Duration, lastCheck time.Time) (map[string]bool, error) {
	now := time.Now().UTC()

	// The heartbeat must be older than the current threshold...
	cutoffNow := now.Add(-offlineThreshold).Format(time.RFC3339)
	// ...but newer than or equal to the threshold was at the time of the last check.
	cutoffPrev := lastCheck.UTC().Add(-offlineThreshold).Format(time.RFC3339)

	updatedNodes := make(map[string]bool)

	// 1. Identify nodes that have gone offline
	rows, err := s.DB.Query(`
		SELECT id FROM nodes 
		WHERE last_heartbeat < ? AND last_heartbeat >= ? AND last_heartbeat != ''
	`, cutoffNow, cutoffPrev)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err == nil {
				updatedNodes[id] = true
			}
		}
	} else {
		return nil, err
	}

	// 2. Identify sensors that have gone offline
	sensorRows, err := s.DB.Query(`
		SELECT node_id FROM node_sensors 
		WHERE last_heartbeat < ? AND last_heartbeat >= ? AND last_heartbeat != ''
	`, cutoffNow, cutoffPrev)
	if err == nil {
		defer sensorRows.Close()
		for sensorRows.Next() {
			var nid string
			if err := sensorRows.Scan(&nid); err == nil {
				updatedNodes[nid] = true
			}
		}
	} else {
		return nil, err
	}

	return updatedNodes, nil
}
