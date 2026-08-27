package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrations_BackupCreationOnUpgrade(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_upgrade.db")

	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("Failed to open sqlite: %v", err)
	}

	// 1. Manually setup v1 schema and records
	_, err = db.Exec(`
		CREATE TABLE schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL
		);
		INSERT INTO schema_migrations (version, applied_at) VALUES (1, '2026-01-01T00:00:00Z');

		CREATE TABLE nodes (
			id TEXT PRIMARY KEY,
			alias TEXT NOT NULL,
			api_key TEXT UNIQUE NOT NULL,
			public_ip TEXT,
			private_ip TEXT,
			tags TEXT NOT NULL DEFAULT '[]',
			pending_config INTEGER NOT NULL DEFAULT 0,
			active_revision TEXT,
			desired_revision TEXT,
			last_heartbeat TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);

		CREATE TABLE node_sensors (
			node_id TEXT NOT NULL,
			sensor_id TEXT NOT NULL, 
			custom_name TEXT NOT NULL, 
			config_values TEXT NOT NULL DEFAULT '{}',
			config_rev TEXT NOT NULL DEFAULT '',
			deployed_version TEXT NOT NULL DEFAULT '',
			last_heartbeat TEXT,
			is_silenced INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (node_id, sensor_id),
			FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE
		);

		CREATE TABLE sensor_heartbeats (
			node_id TEXT NOT NULL,
			sensor_id TEXT NOT NULL,
			time_bucket TEXT NOT NULL,
			PRIMARY KEY (node_id, sensor_id, time_bucket),
			FOREIGN KEY (node_id, sensor_id) REFERENCES node_sensors(node_id, sensor_id) ON DELETE CASCADE
		);

		INSERT INTO nodes (id, alias, api_key, created_at, updated_at) VALUES ('n1', 'Node 1', 'key1', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO node_sensors (node_id, sensor_id, custom_name, created_at, updated_at) VALUES ('n1', 's1', 'Sensor 1', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO sensor_heartbeats (node_id, sensor_id, time_bucket) VALUES ('n1', 's1', '2026-01-01T10:00:00Z');
		INSERT INTO sensor_heartbeats (node_id, sensor_id, time_bucket) VALUES ('n1', 's1', '2026-01-01T10:01:00Z');
	`)
	if err != nil {
		t.Fatalf("Failed to seed v1 database: %v", err)
	}

	// 2. Run Migrations 1 -> 2 with dbPath passed
	if err := RunMigrations(db, dbPath); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// 3. Verify that a backup file was created on disk
	matches, err := filepath.Glob(filepath.Join(tmpDir, "test_upgrade.db.pre-v1-*.bak"))
	if err != nil || len(matches) == 0 {
		t.Fatalf("Expected pre-migration backup file matching test_upgrade.db.pre-v1-*.bak, found %d: %v", len(matches), err)
	}
	backupFile := matches[0]

	// 4. Verify backup file contains original v1 table
	backupDb, err := sql.Open("sqlite", backupFile)
	if err != nil {
		t.Fatalf("Failed to open backup database: %v", err)
	}
	defer backupDb.Close()

	var count int
	err = backupDb.QueryRow("SELECT COUNT(*) FROM sensor_heartbeats").Scan(&count)
	if err != nil {
		t.Fatalf("Backup database should contain sensor_heartbeats table: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 heartbeats in backup DB, got %d", count)
	}

	var backupVersion int
	err = backupDb.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&backupVersion)
	if err != nil || backupVersion != 1 {
		t.Errorf("Expected backup DB schema version 1, got %d (err: %v)", backupVersion, err)
	}

	// 5. Verify live database contains migrated v2 state
	var liveVersion int
	err = db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&liveVersion)
	if err != nil || liveVersion != 2 {
		t.Errorf("Expected live DB schema version 2, got %d (err: %v)", liveVersion, err)
	}

	var statusCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sensor_status_changes").Scan(&statusCount)
	if err != nil || statusCount == 0 {
		t.Errorf("Expected migrated status changes in live DB, got %d (err: %v)", statusCount, err)
	}

	_ = db.Close()
	_ = os.Remove(dbPath)
}

func TestMigrations_AtomicRollbackOnFailure(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_rollback.db")

	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("Failed to open sqlite: %v", err)
	}
	defer db.Close()

	// 1. Setup base schema migrations table at version 1
	_, err = db.Exec(`
		CREATE TABLE schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL
		);
		INSERT INTO schema_migrations (version, applied_at) VALUES (1, '2026-01-01T00:00:00Z');
	`)
	if err != nil {
		t.Fatalf("Failed to seed initial state: %v", err)
	}

	// 2. Temporarily inject a failing migration
	origMigrations := migrations
	defer func() { migrations = origMigrations }()

	migrations = append(migrations, Migration{
		Version:     999,
		Description: "Failing test migration",
		Up: func(tx *sql.Tx) error {
			// Do a change then fail
			if _, err := tx.Exec("CREATE TABLE should_be_rolled_back (id INTEGER);"); err != nil {
				return err
			}
			return fmt.Errorf("intentional migration failure for testing rollback")
		},
	})

	// 3. Execute RunMigrations and expect an error
	err = RunMigrations(db, dbPath)
	if err == nil {
		t.Fatalf("Expected RunMigrations to fail on failing migration step, got nil")
	}

	// 4. Verify that the transaction rolled back completely
	var tableExists int
	_ = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='should_be_rolled_back'").Scan(&tableExists)
	if tableExists != 0 {
		t.Errorf("Table 'should_be_rolled_back' should have been rolled back and not exist!")
	}

	var version int
	err = db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version)
	if err != nil || version != 1 {
		t.Errorf("Expected schema_migrations version to remain 1 after rollback, got %d (err: %v)", version, err)
	}
}
