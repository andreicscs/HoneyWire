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
		if err := rows.Scan(&k, &v); err == nil {
			kv[k] = v
		}
	}
	return kv, nil
}

func (s *SQLiteStore) CompleteSetup(adminHash, hubEndpoint, hubKey string) error {
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
	_, err = tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES ('hub_key', ?)", hubKey)
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
		case "hub_endpoint", "hub_key", "webhook_url", "siem_address":
			if strVal, ok := val.(string); ok {
				tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)", key, strVal)
			}
		case "webhook_type":
			if strVal, ok := val.(string); ok && validWebhooks[strings.ToLower(strVal)] {
				tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)", key, strings.ToLower(strVal))
			}
		case "siem_protocol":
			if strVal, ok := val.(string); ok && validProtocols[strings.ToLower(strVal)] {
				tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)", key, strings.ToLower(strVal))
			}
		case "auto_archive_days", "auto_purge_days":
			if numVal, ok := val.(float64); ok {
				tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)", key, strconv.Itoa(int(numVal)))
			}
		case "webhook_events":
			if arrVal, ok := val.([]interface{}); ok {
				var events []string
				for _, v := range arrVal {
					if str, ok := v.(string); ok {
						events = append(events, str)
					}
				}
				tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)", key, strings.Join(events, ","))
			}
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) FactoryReset() error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tx.Exec("DELETE FROM events")
	tx.Exec("DELETE FROM sensors")
	tx.Exec("DELETE FROM sensor_heartbeats")
	tx.Exec("DELETE FROM config")
	
	if err := tx.Commit(); err != nil {
		return err
	}

	return InitializeDefaultConfig(s.DB)
}