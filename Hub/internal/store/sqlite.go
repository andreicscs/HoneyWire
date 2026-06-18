package store

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"
)


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

	if err := RunMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
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
		// TODO: Add "registry_url": "https://raw.githubusercontent.com/andreicscs/HoneyWire/registry-pages" default
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
