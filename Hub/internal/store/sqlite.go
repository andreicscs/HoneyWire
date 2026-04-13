package store

import (
	"database/sql"
	"log"
	"fmt"

	_ "modernc.org/sqlite"
)

const InitSchema = `
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
INSERT OR IGNORE INTO config (key, value) VALUES ('is_armed', 'true');

CREATE TABLE IF NOT EXISTS sensor_heartbeats (
    sensor_id   TEXT NOT NULL,
    time_bucket TEXT NOT NULL,
    PRIMARY KEY (sensor_id, time_bucket)
);
CREATE INDEX IF NOT EXISTS idx_heartbeats_time ON sensor_heartbeats(time_bucket);
`

type Store struct {
	DB *sql.DB
}

// NewStore connects to SQLite, runs the migrations, and returns the Store
func NewStore(dbPath string) (*Store, error) {
	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)


	_, err = db.Exec(InitSchema)
	if err != nil {
		return nil, err
	}

	log.Println("Database initialized successfully in WAL mode.")
	return &Store{DB: db}, nil
}