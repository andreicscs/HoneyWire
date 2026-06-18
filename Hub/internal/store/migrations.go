package store

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

type Migration struct {
	Version     int
	Description string
	Up          func(tx *sql.Tx) error
}

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
    agent_version   TEXT NOT NULL DEFAULT '',
    contract_version TEXT NOT NULL DEFAULT '',
    config_rev      TEXT NOT NULL DEFAULT '',
	deployed_version TEXT NOT NULL DEFAULT '',
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

// WARNING: Schema Migrations & Foreign Keys
//
// HoneyWire uses ON DELETE CASCADE for critical tables (e.g., deleting a node_sensor 
// automatically deletes all its events).
// 
// When altering schemas, NEVER use the legacy SQLite workaround of creating a new table,
// copying data, and dropping the old table. If foreign keys are enabled, dropping the 
// old table will instantly trigger the cascade and permanently delete all dependent rows.
//
// ALWAYS use native SQLite 3.35.0+ ALTER TABLE statements (e.g., ADD COLUMN or DROP COLUMN)
// to mutate schemas safely without destroying dependent data.

var migrations = []Migration{
	{
		Version:     1,
		Description: "Base v2 schema",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(v2Schema)
			return err
		},
	},
}

// RunMigrations applies all pending schema changes transactionally.
func RunMigrations(db *sql.DB) error {
	// 1. Run Integrity Check first to ensure healthy state
	var integrityResult string
	err := db.QueryRow("PRAGMA integrity_check;").Scan(&integrityResult)
	if err != nil || integrityResult != "ok" {
		return fmt.Errorf("database integrity check failed (result: %s): %w", integrityResult, err)
	}

	// 2. Create the tracking table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// 3. Determine current version
	var currentVersionNull sql.NullInt64
	err = db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&currentVersionNull)
	if err != nil {
		return fmt.Errorf("failed to check current migration version: %w", err)
	}

	currentVersion := 0
	if currentVersionNull.Valid {
		currentVersion = int(currentVersionNull.Int64)
	}

	for i := 1; i < len(migrations); i++ {
		if migrations[i].Version <= migrations[i-1].Version {
			return fmt.Errorf(
				"migrations out of order: %d <= %d",
				migrations[i].Version,
				migrations[i-1].Version,
			)
		}
	}

	// 4. Calculate latest target version
	latestVersion := 0
	if len(migrations) > 0 {
		latestVersion = migrations[len(migrations)-1].Version
	}

	log.Printf("[DB] Current schema version: %d", currentVersion)
	log.Printf("[DB] Latest schema version: %d", latestVersion)

	if currentVersion == latestVersion {
		log.Println("[DB] Database up to date.")
		return nil
	}

	if currentVersion > latestVersion {
		return fmt.Errorf(
			"database schema version %d is newer than supported version %d",
			currentVersion,
			latestVersion,
		)
	}

	log.Printf("[DB] Running schema migration %d -> %d", currentVersion, latestVersion)

	// 5. Apply pending migrations inside a single transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start migration transaction: %w", err)
	}
	defer tx.Rollback()

	for _, m := range migrations {
		if m.Version <= currentVersion {
			continue
		}

		log.Printf("[DB] Applying migration %d: %s...", m.Version, m.Description)
		if err := m.Up(tx); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", m.Version, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)", m.Version, time.Now().UTC().Format(time.RFC3339)); err != nil {
			return fmt.Errorf("failed to record migration %d: %w", m.Version, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migrations: %w", err)
	}

	log.Println("[DB] Database up to date.")
	return nil
}
