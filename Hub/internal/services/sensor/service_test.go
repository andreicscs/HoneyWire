package sensor

import (
	"testing"
	"time"
)

type mockStore struct {
	latestStatus string
	insertedStatus []string
	lastHeartbeat string
}

func (m *mockStore) ProcessHeartbeat(nodeID, sensorID, configRev, nowStr string) (bool, error) {
	m.lastHeartbeat = nowStr
	return false, nil
}

func (m *mockStore) MarkSensorOffline(nodeID, sensorID, offlineTime string) error {
	return nil
}

func (m *mockStore) UpdateSensorSilence(nodeID, sensorID string, silenceVal int) error {
	return nil
}

func (m *mockStore) GetTransitionedOfflineNodes(offlineThreshold time.Duration, lastCheck time.Time) (map[string]bool, error) {
	return nil, nil
}

func (m *mockStore) GetSensorLastHeartbeat(nodeID, sensorID string) (string, error) {
	return m.lastHeartbeat, nil
}

func (m *mockStore) GetSensorLatestStatus(nodeID, sensorID string) (string, error) {
	return m.latestStatus, nil
}

func (m *mockStore) InsertStatusChange(nodeID, sensorID, status, timestamp string) error {
	m.insertedStatus = append(m.insertedStatus, status)
	m.latestStatus = status
	return nil
}

type mockBroadcaster struct {
	events []string
}

func (b *mockBroadcaster) Broadcast(topic string, payload interface{}) {
	b.events = append(b.events, topic)
}

func TestProcessHeartbeat_OnlineTransition(t *testing.T) {
	st := &mockStore{latestStatus: "offline"}
	br := &mockBroadcaster{}
	svc := NewService(st, br)

	err := svc.ProcessHeartbeat("node1", "sensor1", "rev1")
	if err != nil {
		t.Fatalf("ProcessHeartbeat failed: %v", err)
	}

	// Should have inserted "online" status change
	if len(st.insertedStatus) != 1 || st.insertedStatus[0] != "online" {
		t.Errorf("Expected 1 'online' status insert, got: %v", st.insertedStatus)
	}

	// Should have broadcasted NODE_SYNCED and SYNC_CHARTS
	hasSyncCharts := false
	for _, e := range br.events {
		if e == "SYNC_CHARTS" {
			hasSyncCharts = true
		}
	}
	if !hasSyncCharts {
		t.Errorf("Expected SYNC_CHARTS broadcast on transition to online")
	}

	// Second heartbeat when already online should NOT insert a duplicate status change
	err = svc.ProcessHeartbeat("node1", "sensor1", "rev1")
	if err != nil {
		t.Fatalf("Second ProcessHeartbeat failed: %v", err)
	}
	if len(st.insertedStatus) != 1 {
		t.Errorf("Expected status changes count to remain 1, got %d", len(st.insertedStatus))
	}
}
