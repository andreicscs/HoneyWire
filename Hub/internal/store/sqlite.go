package store

import (
	"database/sql"
	"fmt"
	"log"
	"time"

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
	// Enable WAL mode, set a 5-second busy timeout to prevent locking, and enable Foreign Keys
	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)", dbPath)
	
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// Connection Pooling for SQLite performance
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Apply the clean v2 schema
	if _, err := db.Exec(v2Schema); err != nil {
		return nil, err
	}

	if err := InitializeDefaultConfig(db); err != nil {
		return nil, fmt.Errorf("failed to initialize default config: %w", err)
	}

	log.Println("[DB] Database v2.0.0 initialized successfully in WAL mode.")
	return &Store{DB: db}, nil
}

func InitializeDefaultConfig(db *sql.DB) error {
	defaults := map[string]string{
		"is_armed":          "true",
		"webhook_type":      "none",
		"webhook_url":       "",
		"webhook_events":    "[]",
		"auto_archive_days": "90",
		"auto_purge_days":   "180",
		"siem_address":      "",
		"siem_protocol":     "syslog",
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT OR IGNORE INTO config (key, value) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for k, v := range defaults {
		if _, err := stmt.Exec(k, v); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}