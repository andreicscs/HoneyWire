package store

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

const v2Schema = `
-- Nodes: Physical servers/agents managing sensors
CREATE TABLE IF NOT EXISTS nodes (
    node_id     TEXT PRIMARY KEY,
    alias       TEXT NOT NULL,
    node_key    TEXT UNIQUE NOT NULL,
    ip_address  TEXT,
    first_seen  TEXT NOT NULL,
    last_seen   TEXT NOT NULL
);

-- Sensors: Monitored honeypots, strictly owned by a Node
CREATE TABLE IF NOT EXISTS sensors (
    node_id     TEXT NOT NULL,
    sensor_id   TEXT NOT NULL,
    first_seen  TEXT NOT NULL,
    last_seen   TEXT NOT NULL,
    metadata    TEXT NOT NULL DEFAULT '{}',
    is_silenced INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (node_id, sensor_id),
    FOREIGN KEY (node_id) REFERENCES nodes(node_id) ON DELETE CASCADE
);

-- Events: Security alerts from sensors
CREATE TABLE IF NOT EXISTS events (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id         TEXT NOT NULL,
    sensor_id       TEXT NOT NULL,
    timestamp       TEXT NOT NULL,
    contract_version TEXT NOT NULL DEFAULT '1.0.0',
    event_trigger   TEXT NOT NULL DEFAULT 'alert',
    severity        TEXT NOT NULL DEFAULT 'medium',
    source          TEXT NOT NULL DEFAULT 'Unknown',
    target          TEXT NOT NULL DEFAULT 'Unknown',
    details         TEXT NOT NULL DEFAULT '{}',
    is_read         INTEGER NOT NULL DEFAULT 0,
    is_archived     INTEGER NOT NULL DEFAULT 0,
    count           INTEGER NOT NULL DEFAULT 1,
    FOREIGN KEY (node_id, sensor_id) REFERENCES sensors(node_id, sensor_id) ON DELETE CASCADE
);

-- Sensor Heartbeats: Routine health pings from sensors
CREATE TABLE IF NOT EXISTS sensor_heartbeats (
    node_id     TEXT NOT NULL,
    sensor_id   TEXT NOT NULL,
    time_bucket TEXT NOT NULL,
    PRIMARY KEY (node_id, sensor_id, time_bucket),
    FOREIGN KEY (node_id, sensor_id) REFERENCES sensors(node_id, sensor_id) ON DELETE CASCADE
);

-- Pairing Tokens: One-time tokens for secure node provisioning via wizard
CREATE TABLE IF NOT EXISTS pairing_tokens (
    token       TEXT PRIMARY KEY,
    expires_at  DATETIME NOT NULL,
    created_at  DATETIME NOT NULL
);

-- System Configuration
CREATE TABLE IF NOT EXISTS config (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- High-performance indexes for dashboard queries
CREATE INDEX IF NOT EXISTS idx_events_archived ON events(is_archived, id DESC);
CREATE INDEX IF NOT EXISTS idx_events_node_sensor ON events(node_id, sensor_id);
CREATE INDEX IF NOT EXISTS idx_events_severity ON events(severity);
CREATE INDEX IF NOT EXISTS idx_sensors_node ON sensors(node_id);
CREATE INDEX IF NOT EXISTS idx_heartbeats_time ON sensor_heartbeats(time_bucket);
CREATE INDEX IF NOT EXISTS idx_pairing_tokens_expires ON pairing_tokens(expires_at);
`

type Store struct {
	DB *sql.DB
}

// NewStore initializes the SQLite database with the v2.0.0 schema
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, err
	}

	// Apply the clean v2 schema
	if _, err := db.Exec(v2Schema); err != nil {
		return nil, err
	}

	return &Store{DB: db}, nil
}

func InitializeDefaultConfig(db *sql.DB) error {
	// Check if config already exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM config").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil // Already initialized
	}

	// Insert default config values
	defaultConfig := map[string]string{
		"is_armed":          "true",
		"webhook_type":      "none",
		"webhook_url":       "",
		"webhook_events":    "[]",
		"auto_archive_days": "90",
		"auto_purge_days":   "180",
		"siem_address":      "",
		"siem_protocol":     "syslog",
	}

	for key, value := range defaultConfig {
		_, err := db.Exec("INSERT INTO config (key, value) VALUES (?, ?)", key, value)
		if err != nil {
			return err
		}
	}

	return nil
}
