package store

import (
	"strconv"
	"strings"
)

func (s *SQLiteStore) GetConfigValue(key string) (string, error) {
	var value string
	err := s.DB.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&value)
	return value, err
}

func (s *SQLiteStore) UpdateConfigValue(key, value string) error {
	_, err := s.DB.Exec("INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)", key, value)
	return err
}

func (s *SQLiteStore) GetAllConfig() (map[string]string, error) {
	rows, err := s.DB.Query("SELECT key, value FROM config")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	kv := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		kv[k] = v
	}
	return kv, nil
}

func (s *SQLiteStore) CompleteSetup(adminHash, hubEndpoint string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES ('admin_hash', ?)", adminHash)
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES ('hub_endpoint', ?)", hubEndpoint)
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES ('is_setup', 'true')")
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLiteStore) UpdateConfigBatch(req map[string]interface{}) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	validWebhooks := map[string]bool{"ntfy": true, "gotify": true, "discord": true, "slack": true}
	validProtocols := map[string]bool{"tcp": true, "udp": true}

	for key, val := range req {
		switch key {
		case "hub_endpoint", "webhook_url", "siem_address", "registry_url", "whitelisted_sources":
			if strVal, ok := val.(string); ok {
				if _, err := tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)", key, strVal); err != nil {
					return err
				}
			}
		case "webhook_type":
			if strVal, ok := val.(string); ok && validWebhooks[strings.ToLower(strVal)] {
				if _, err := tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)", key, strings.ToLower(strVal)); err != nil {
					return err
				}
			}
		case "siem_protocol":
			if strVal, ok := val.(string); ok && validProtocols[strings.ToLower(strVal)] {
				if _, err := tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)", key, strings.ToLower(strVal)); err != nil {
					return err
				}
			}
		case "auto_archive_days", "auto_purge_days":
			if numVal, ok := val.(float64); ok {
				if _, err := tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)", key, strconv.Itoa(int(numVal))); err != nil {
					return err
				}
			}
		case "webhook_events":
			if arrVal, ok := val.([]interface{}); ok {
				var events []string
				for _, v := range arrVal {
					if str, ok := v.(string); ok {
						events = append(events, str)
					}
				}
				if _, err := tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)", key, strings.Join(events, ",")); err != nil {
					return err
				}
			}
		}
	}

	return tx.Commit()
}

// FactoryReset completely wipes the database back to a blank slate
func (s *SQLiteStore) FactoryReset() error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Rollback on any error.

	// Wipe all tables. The order is important for manual deletion without cascades.
	// It is best practice to be explicit, even though ON DELETE CASCADE is enabled.
	queries := []string{
		"DELETE FROM events",
		"DELETE FROM sensor_heartbeats",
		"DELETE FROM node_sensors",
		"DELETE FROM nodes",
		"DELETE FROM config",
		// Reset the autoincrement counter for the events table.
		"DELETE FROM sqlite_sequence WHERE name = 'events'",
	}

	for _, q := range queries {
		if _, err := tx.Exec(q); err != nil {
			return err
		}
	}

	// Re-initialize default config within the same transaction.
	if err := initializeDefaultConfigTx(tx); err != nil {
		return err
	}

	return tx.Commit()
}

// FactoryResetDryRun calculates the number of rows that would be deleted
func (s *SQLiteStore) FactoryResetDryRun() (map[string]int, error) {
	stats := make(map[string]int)

	queries := map[string]string{
		"events":            "SELECT COUNT(*) FROM events",
		"sensor_heartbeats": "SELECT COUNT(*) FROM sensor_heartbeats",
		"node_sensors":      "SELECT COUNT(*) FROM node_sensors",
		"nodes":             "SELECT COUNT(*) FROM nodes",
	}

	for key, query := range queries {
		var count int
		if err := s.DB.QueryRow(query).Scan(&count); err != nil {
			return nil, err
		}
		stats[key] = count
	}

	return stats, nil
}

