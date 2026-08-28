package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
)

func TestGetSensorsForUptime_FirstSeenPreservedAfterPurge(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_sensors.db")

	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("Failed to open sqlite: %v", err)
	}
	defer db.Close()

	if err := RunMigrations(db, dbPath); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	st := &SQLiteStore{DB: db}

	// 1. Seed node and sensor created 60 days ago
	createdAt := "2026-06-01T00:00:00Z"
	_, err = db.Exec(`
		INSERT INTO nodes (id, alias, api_key, created_at, updated_at)
		VALUES ('n1', 'Node 1', 'key1', ?, ?);
		INSERT INTO node_sensors (node_id, sensor_id, custom_name, created_at, updated_at)
		VALUES ('n1', 's1', 'Sensor 1', ?, ?);
	`, createdAt, createdAt, createdAt, createdAt)
	if err != nil {
		t.Fatalf("Failed to seed node & sensor: %v", err)
	}

	// 2. Add status changes where the oldest retained is only from yesterday
	yesterday := "2026-07-30T12:00:00Z"
	_, err = db.Exec(`
		INSERT INTO sensor_status_changes (node_id, sensor_id, status, timestamp)
		VALUES ('n1', 's1', 'online', ?);
	`, yesterday)
	if err != nil {
		t.Fatalf("Failed to insert status change: %v", err)
	}

	// 3. Call GetSensorsForUptime
	sensors, err := st.GetSensorsForUptime("2026-07-31T00:00:00Z")
	if err != nil {
		t.Fatalf("GetSensorsForUptime failed: %v", err)
	}

	if len(sensors) != 1 {
		t.Fatalf("Expected 1 sensor, got %d", len(sensors))
	}

	// 4. Verify FirstSeen reflects created_at (2026-06-01), NOT yesterday's status change
	if sensors[0].FirstSeen != createdAt {
		t.Errorf("Expected FirstSeen to be %s (sensor creation date), got %s", createdAt, sensors[0].FirstSeen)
	}
}
