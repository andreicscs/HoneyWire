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
    id              TEXT PRIMARY KEY,
    alias           TEXT NOT NULL,
    api_key         TEXT UNIQUE NOT NULL,
    public_ip       TEXT,
    private_ip      TEXT,
    tags            TEXT NOT NULL DEFAULT '[]',
    pending_config    INTEGER NOT NULL DEFAULT 0,
    active_revision   TEXT,
    desired_revision  TEXT,
    last_heartbeat    TEXT,
    created_at        TEXT NOT NULL,
    updated_at        TEXT NOT NULL
);

-- NodeSensors: Installed instances of catalog sensors
CREATE TABLE IF NOT EXISTS node_sensors (
    node_id         TEXT NOT NULL,
    sensor_id       TEXT NOT NULL, 
    custom_name     TEXT NOT NULL, 
    config_values   TEXT NOT NULL DEFAULT '{}',
    metadata        TEXT NOT NULL DEFAULT '{}',
	last_heartbeat  TEXT,
    is_silenced     INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    PRIMARY KEY (node_id, sensor_id),
    FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE
);

-- Events: Security alerts from sensors
CREATE TABLE IF NOT EXISTS events (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id          TEXT NOT NULL,
    sensor_id        TEXT NOT NULL, 
    timestamp        TEXT NOT NULL,
    contract_version TEXT NOT NULL DEFAULT '1.0.0',
    event_trigger    TEXT NOT NULL DEFAULT 'alert',
    severity         TEXT NOT NULL DEFAULT 'medium',
    source           TEXT NOT NULL DEFAULT 'Unknown',
    target           TEXT NOT NULL DEFAULT 'Unknown',
    details          TEXT NOT NULL DEFAULT '{}',
    is_read          INTEGER NOT NULL DEFAULT 0,
    is_archived      INTEGER NOT NULL DEFAULT 0,
    count            INTEGER NOT NULL DEFAULT 1,
    FOREIGN KEY (node_id, sensor_id) REFERENCES node_sensors(node_id, sensor_id) ON DELETE CASCADE
);

-- Sensor Heartbeats: Routine health pings from sensors
CREATE TABLE IF NOT EXISTS sensor_heartbeats (
    node_id     TEXT NOT NULL,
    sensor_id   TEXT NOT NULL,
    time_bucket TEXT NOT NULL,
    PRIMARY KEY (node_id, sensor_id, time_bucket),
    FOREIGN KEY (node_id, sensor_id) REFERENCES node_sensors(node_id, sensor_id) ON DELETE CASCADE
);

-- System Configuration
CREATE TABLE IF NOT EXISTS config (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- High-performance indexes
CREATE INDEX IF NOT EXISTS idx_events_archived ON events(is_archived, id DESC);
CREATE INDEX IF NOT EXISTS idx_events_node_sensor ON events(node_id, sensor_id);
CREATE INDEX IF NOT EXISTS idx_events_severity ON events(severity);
CREATE INDEX IF NOT EXISTS idx_sensors_node ON node_sensors(node_id);
CREATE INDEX IF NOT EXISTS idx_heartbeats_time ON sensor_heartbeats(time_bucket);
`

type SQLiteStore struct {
	DB *sql.DB
}

func NewStore(dbPath string) (*SQLiteStore, error) {
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

	if _, err := db.Exec(v2Schema); err != nil {
		return nil, err
	}

	if err := InitializeDefaultConfig(db); err != nil {
		return nil, fmt.Errorf("failed to initialize default config: %w", err)
	}

	log.Println("[DB] Database v2.0.0 initialized successfully in WAL mode.")
	return &SQLiteStore{DB: db}, nil
}

func InitializeDefaultConfig(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := initializeDefaultConfigTx(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func initializeDefaultConfigTx(tx *sql.Tx) error {
	defaults := map[string]string{
		"is_armed":          "true",
		"webhook_type":      "ntfy",
		"webhook_url":       "",
		"webhook_events":    "critical,high,medium,low",
		"auto_archive_days": "0",
		"auto_purge_days":   "0",
		"siem_address":      "",
		"siem_protocol":     "tcp",
	}

	stmt, err := tx.Prepare("INSERT OR IGNORE INTO config (key, value) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for k, v := range defaults {
		if _, err := stmt.Exec(k, v); err != nil {
			return err
		}
	}

	return nil
}
