package store

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

// baselineSchema represents v1 of the database
const baselineSchema = `
CREATE TABLE IF NOT EXISTS events (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp        TEXT    NOT NULL,
    contract_version TEXT    NOT NULL DEFAULT '1.0.0',
    sensor_id        TEXT    NOT NULL,
    event_trigger    TEXT    NOT NULL DEFAULT 'alert',
    severity         TEXT    NOT NULL DEFAULT 'medium',
    source           TEXT    NOT NULL DEFAULT 'Unknown',
    target           TEXT    NOT NULL DEFAULT 'Unknown',
    details          TEXT    NOT NULL DEFAULT '{}',
    is_read          INTEGER NOT NULL DEFAULT 0,
    is_archived      INTEGER NOT NULL DEFAULT 0,
    count            INTEGER NOT NULL DEFAULT 1
);

-- HIGH PERFORMANCE INDEXES FOR DASHBOARD QUERIES
CREATE INDEX IF NOT EXISTS idx_events_archived ON events(is_archived, id DESC);
CREATE INDEX IF NOT EXISTS idx_events_sensor ON events(sensor_id);
CREATE INDEX IF NOT EXISTS idx_events_severity ON events(severity);

CREATE TABLE IF NOT EXISTS sensors (
    sensor_id   TEXT PRIMARY KEY,
    first_seen  TEXT,
    last_seen   TEXT NOT NULL,
    metadata    TEXT NOT NULL DEFAULT '{}',
    is_silenced INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS config (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sensor_heartbeats (
    sensor_id   TEXT NOT NULL,
    time_bucket TEXT NOT NULL,
    PRIMARY KEY (sensor_id, time_bucket)
);
CREATE INDEX IF NOT EXISTS idx_heartbeats_time ON sensor_heartbeats(time_bucket);
`

// migrations holds all schema changes in chronological order.
var migrations = []string{
	baselineSchema, // Version 1 (v1.0.0)
}

type Store struct {
	DB *sql.DB
}

// backupDB creates a copy of the SQLite file before applying destructive changes
func backupDB(dbPath string, version int) {
	if version == 0 {
		return // Nothing to back up on a completely fresh install
	}

	backupPath := fmt.Sprintf("%s.v%d.bak", dbPath, version)
	
	sourceFile, err := os.Open(dbPath)
	if err != nil {
		log.Printf("[!] Warning: Could not open DB for backup: %v", err)
		return
	}
	defer sourceFile.Close()

	destFile, err := os.Create(backupPath)
	if err != nil {
		log.Printf("[!] Warning: Could not create DB backup file: %v", err)
		return
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err == nil {
		log.Printf("[DB] Auto-Backup created at %s before applying migrations.", backupPath)
	}
}

// runMigrations safely advances the database schema to the latest version
func runMigrations(db *sql.DB, dbPath string) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY)`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	var currentVersion int
	err = db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("failed to read current schema version: %w", err)
	}

	// If we are about to apply new migrations to an existing DB, back it up first.
	if currentVersion > 0 && currentVersion < len(migrations) {
		backupDB(dbPath, currentVersion)
	}

	for i := currentVersion; i < len(migrations); i++ {
		targetVersion := i + 1
		log.Printf("[DB] Applying database migration v%d...", targetVersion)

		tx, err := db.Begin()
		if err != nil {
			return err
		}

		if _, err := tx.Exec(migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to apply migration v%d: %v", targetVersion, err)
		}

		if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, targetVersion); err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}
		log.Printf("[DB] Migration v%d applied successfully.", targetVersion)
	}

	return nil
}

func NewStore(dbPath string) (*Store, error) {
	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Pass the dbPath to the migration runner so it knows what file to copy
	if err := runMigrations(db, dbPath); err != nil {
		return nil, fmt.Errorf("database migration failed: %w", err)
	}

	if err := InitializeDefaultConfig(db); err != nil {
		return nil, fmt.Errorf("failed to initialize default config: %w", err)
	}

	log.Println("[DB] Database initialized successfully in WAL mode.")
	return &Store{DB: db}, nil
}

func InitializeDefaultConfig(db *sql.DB) error {
	defaults := map[string]string{
		"is_armed":          "true",
		"is_setup":          "false",
		"hub_endpoint":      "",
		"hub_key":           "",
		"auto_archive_days": "0",
		"auto_purge_days":   "0",
		"webhook_url":       "",
		"webhook_type":      "ntfy",
		"webhook_events":    "critical,high,medium,low,info",
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