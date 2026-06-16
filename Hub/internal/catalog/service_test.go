package catalog_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/honeywire/hub/internal/catalog"
)

type MockStore struct {
	RegistryURL string
}

func (m *MockStore) GetConfigValue(key string) (string, error) {
	if key == "registry_url" {
		return m.RegistryURL, nil
	}
	return "", nil
}

type MockBroadcaster struct {
	BroadcastCount int
}

func (m *MockBroadcaster) Broadcast(eventType string, payload interface{}) {
	if eventType == "CATALOG_UPDATED" {
		m.BroadcastCount++
	}
}

func TestRefreshIndexDeduplication(t *testing.T) {
	responseIndex := map[string]interface{}{
		"sensors": []map[string]interface{}{
			{
				"id":     "hw-sensor-test",
				"latest": "v1.0.0",
				"versions": []map[string]interface{}{
					{"v": "v1.0.0", "min_hub_version": "v1.0.0"},
				},
			},
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(responseIndex)
	}))
	defer ts.Close()

	store := &MockStore{RegistryURL: ts.URL}
	broadcaster := &MockBroadcaster{}
	svc := catalog.NewService(store, broadcaster)

	// First fetch should broadcast
	if err := svc.RefreshIndex(); err != nil {
		t.Fatalf("First RefreshIndex failed: %v", err)
	}

	if broadcaster.BroadcastCount != 1 {
		t.Errorf("Expected exactly 1 broadcast on initial pull, got %d", broadcaster.BroadcastCount)
	}

	// Second fetch with IDENTICAL data should NOT broadcast
	if err := svc.RefreshIndex(); err != nil {
		t.Fatalf("Second RefreshIndex failed: %v", err)
	}

	if broadcaster.BroadcastCount != 1 {
		t.Errorf("Expected broadcast count to remain 1 on identical pull, got %d", broadcaster.BroadcastCount)
	}

	// Third fetch with CHANGED data SHOULD broadcast
	responseIndex["sensors"].([]map[string]interface{})[0]["latest"] = "v2.0.0"

	if err := svc.RefreshIndex(); err != nil {
		t.Fatalf("Third RefreshIndex failed: %v", err)
	}

	if broadcaster.BroadcastCount != 2 {
		t.Errorf("Expected broadcast count to increment to 2 on changed pull, got %d", broadcaster.BroadcastCount)
	}
}

func TestRefreshIndexTriggersHook(t *testing.T) {
	responseIndex := map[string]interface{}{
		"sensors": []map[string]interface{}{
			{
				"id":      "hw-sensor-test",
				"latest": "v1.0.0",
			},
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(responseIndex)
	}))
	defer ts.Close()

	store := &MockStore{RegistryURL: ts.URL}
	broadcaster := &MockBroadcaster{}
	svc := catalog.NewService(store, broadcaster)

	var mu sync.Mutex
	hookFiredCount := 0
	svc.SetOnChangeHook(func() {
		mu.Lock()
		hookFiredCount++
		mu.Unlock()
	})

	if err := svc.RefreshIndex(); err != nil {
		t.Fatalf("First RefreshIndex failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	count := hookFiredCount
	mu.Unlock()
	if count != 1 {
		t.Errorf("Expected hook to fire exactly 1 time on initial pull, got %d", count)
	}

	// Second fetch with IDENTICAL data should NOT trigger hook
	if err := svc.RefreshIndex(); err != nil {
		t.Fatalf("Second RefreshIndex failed: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	count = hookFiredCount
	mu.Unlock()
	if count != 1 {
		t.Errorf("Expected hook count to remain 1 on identical pull, got %d", count)
	}

	// Third fetch with CHANGED data SHOULD trigger hook
	responseIndex["sensors"].([]map[string]interface{})[0]["latest"] = "v2.0.0"

	if err := svc.RefreshIndex(); err != nil {
		t.Fatalf("Third RefreshIndex failed: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	count = hookFiredCount
	mu.Unlock()
	if count != 2 {
		t.Errorf("Expected hook count to increment to 2 on changed pull, got %d", count)
	}
}
