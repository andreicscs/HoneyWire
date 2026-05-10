package store

import (
	"database/sql"
	"encoding/json"

	"github.com/honeywire/hub/internal/models"
)

func (s *SQLiteStore) InsertEvent(e *models.Event, nowStr string, detailsStr string) (int, error) {
	result, err := s.DB.Exec(`
		INSERT INTO events (node_id, sensor_id, timestamp, contract_version, event_trigger, severity, source, target, details, is_read, is_archived)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0)`,
		e.NodeID, e.SensorID, nowStr, e.ContractVersion, e.EventTrigger, e.Severity, e.Source, e.Target, detailsStr,
	)
	if err != nil {
		return 0, err
	}
	lastInsertID, _ := result.LastInsertId()
	return int(lastInsertID), nil
}

func (s *SQLiteStore) UpdateNodeLastSeen(nodeID, timestamp string) error {
	_, err := s.DB.Exec(`UPDATE nodes SET last_seen = ? WHERE node_id = ?`, timestamp, nodeID)
	return err
}

func (s *SQLiteStore) IsSensorSilenced(nodeID, sensorID string) (bool, error) {
	var isSilencedInt int
	err := s.DB.QueryRow(
		"SELECT is_silenced FROM sensors WHERE node_id = ? AND sensor_id = ?",
		nodeID, sensorID,
	).Scan(&isSilencedInt)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return isSilencedInt == 1, nil
}

func (s *SQLiteStore) GetEvents(isArchived int, nodeID string, sensorID string) ([]models.Event, error) {
	query := "SELECT id, timestamp, contract_version, sensor_id, node_id, event_trigger, severity, source, target, details, is_read, is_archived FROM events WHERE is_archived = ?"
	args := []interface{}{isArchived}

	// Apply Node Filter if present
	if nodeID != "" {
		query += " AND node_id = ?"
		args = append(args, nodeID)
	}

	// Apply Sensor Filter if present
	if sensorID != "" {
		query += " AND sensor_id = ?"
		args = append(args, sensorID)
	}

	query += " ORDER BY id DESC"

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var e models.Event
		var detailsStr string
		var isReadInt, isArchivedInt int
		var dbNodeID *string // Use pointer to handle SQL NULL safely

		if err := rows.Scan(
			&e.ID, &e.Timestamp, &e.ContractVersion, &e.SensorID, &dbNodeID,
			&e.EventTrigger, &e.Severity, &e.Source, &e.Target,
			&detailsStr, &isReadInt, &isArchivedInt,
		); err != nil {
			continue
		}

		if dbNodeID != nil {
			e.NodeID = *dbNodeID
		}

		e.IsRead = isReadInt == 1
		e.IsArchived = isArchivedInt == 1
		json.Unmarshal([]byte(detailsStr), &e.Details)
		events = append(events, e)
	}

	if events == nil {
		events = []models.Event{}
	}

	return events, nil
}

func (s *SQLiteStore) GetUnreadEventCount() (int, error) {
	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM events WHERE is_read = 0 AND is_archived = 0").Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *SQLiteStore) MarkEventRead(eventID string) error {
	_, err := s.DB.Exec("UPDATE events SET is_read = 1 WHERE id = ?", eventID)
	return err
}

func (s *SQLiteStore) MarkAllEventsRead() error {
	_, err := s.DB.Exec("UPDATE events SET is_read = 1 WHERE is_read = 0")
	return err
}

func (s *SQLiteStore) ArchiveEvent(eventID string) error {
	_, err := s.DB.Exec("UPDATE events SET is_archived = 1, is_read = 1 WHERE id = ?", eventID)
	return err
}

func (s *SQLiteStore) ArchiveAllEvents() error {
	_, err := s.DB.Exec("UPDATE events SET is_archived = 1, is_read = 1 WHERE is_archived = 0")
	return err
}

func (s *SQLiteStore) GetEventCount() (int, error) {
	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM events").Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *SQLiteStore) ClearAllEvents() error {
	_, err := s.DB.Exec("DELETE FROM events")
	return err
}
