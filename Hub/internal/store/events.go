package store

import (
	"database/sql"
	"encoding/json"
	"time"

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
	lastInsertID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(lastInsertID), nil
}

func (s *SQLiteStore) IsSensorSilenced(nodeID, sensorID string) (bool, error) {
	var isSilencedInt int
	err := s.DB.QueryRow(
		"SELECT is_silenced FROM node_sensors WHERE node_id = ? AND sensor_id = ?",
		nodeID, sensorID,
	).Scan(&isSilencedInt)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // Default to not silenced if somehow not found
		}
		return false, err
	}
	return isSilencedInt == 1, nil
}

func (s *SQLiteStore) GetEvents(isArchived int, nodeID string, sensorID string) ([]models.Event, error) {
	query := "SELECT id, timestamp, contract_version, sensor_id, node_id, event_trigger, severity, source, target, details, is_read, is_archived FROM events WHERE is_archived = ?"
	args := []interface{}{isArchived}

	if nodeID != "" {
		query += " AND node_id = ?"
		args = append(args, nodeID)
	}
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

		if err := rows.Scan(
			&e.ID, &e.Timestamp, &e.ContractVersion, &e.SensorID, &e.NodeID,
			&e.EventTrigger, &e.Severity, &e.Source, &e.Target,
			&detailsStr, &isReadInt, &isArchivedInt,
		); err != nil {
			return nil, err
		}

		e.IsRead = isReadInt == 1
		e.IsArchived = isArchivedInt == 1
		if detailsStr != "" {
			e.Details = json.RawMessage(detailsStr)
		} else {
			e.Details = json.RawMessage("{}")
		}
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
	return count, err
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
	return count, err
}

func (s *SQLiteStore) ClearAllEvents() error {
	_, err := s.DB.Exec("DELETE FROM events")
	return err
}

// EnforceRetention automatically archives and deletes old events based on configured days
func (s *SQLiteStore) EnforceRetention(archiveDays, purgeDays int) error {
	now := time.Now().UTC()
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Auto-Archive (if enabled)
	if archiveDays > 0 {
		archiveCutoff := now.AddDate(0, 0, -archiveDays).Format(time.RFC3339)
		if _, err := tx.Exec("UPDATE events SET is_archived = 1 WHERE timestamp < ? AND is_archived = 0", archiveCutoff); err != nil {
			return err
		}
	}

	// 2. Auto-Purge (if enabled)
	if purgeDays > 0 {
		purgeCutoff := now.AddDate(0, 0, -purgeDays).Format(time.RFC3339)
		if _, err := tx.Exec("DELETE FROM events WHERE timestamp < ?", purgeCutoff); err != nil {
			return err
		}
	}

	return tx.Commit()
}
