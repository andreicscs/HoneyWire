package composesvc_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/honeywire/hub/internal/catalog"
	"github.com/honeywire/hub/internal/models"
	composesvc "github.com/honeywire/hub/internal/services/compose"
)

type MockStore struct {
	RegistryURL string
}

func (m *MockStore) GetConfigValue(key string) (string, error) {
	if key == "registry_url" {
		return m.RegistryURL, nil
	}
	if key == "hub_endpoint" {
		return "http://localhost:8080", nil
	}
	return "", nil
}

func (m *MockStore) GetNodeByKey(token string) (string, error) {
	return "node-1", nil
}

func (m *MockStore) GetNodeDetails(nodeID string) (*models.Node, error) {
	return &models.Node{
		ID: "node-1",
		InstalledSensors: []models.NodeSensor{
			{ID: "hw-sensor-test"},
		},
	}, nil
}

func (m *MockStore) SetNodeDesiredRevision(nodeID, rev string) error {
	return nil
}

func (m *MockStore) SetNodeSensorDeployedVersion(nodeID, sensorID, version string) error {
	return nil
}

func (m *MockStore) ApplyNodeRevision(nodeID, revision string) error {
	return nil
}

func TestComposeSmartVersionSelection(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.json" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"sensors": []map[string]interface{}{
					{
						"id":     "hw-sensor-test",
						"latest": "2.0.0",
						"versions": []map[string]string{
							{"v": "1.0.0", "min_hub_api": "1"},
							{"v": "1.5.0", "min_hub_api": " 2 "}, // Injecting malicious whitespace
							{"v": "2.0.0", "min_hub_api": "  3"}, // Injecting malicious whitespace
						},
					},
				},
			})
			return
		}

		// Mock responses for the requested versions
		version := r.URL.Path[len("/test-v") : len(r.URL.Path)-5] // Extract version from path
		
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":             "hw-sensor-test",
			"version":        version,
			"schema_version": "1.0",
			"min_hub_api":    "1", // Mock doesn't need to match perfectly, just needs to parse
			"deployment": map[string]interface{}{
				"image_repository": "test",
				"image_tag":        version,
			},
		})
	}))
	defer ts.Close()

	store := &MockStore{RegistryURL: ts.URL}
	catSvc := catalog.NewService(store)
	svc := composesvc.NewService(store, catSvc)

	// VERSIONING ARCHITECTURE EXPLANATION (SENSOR REGISTRY):
	// The Compose service automatically parses the registry's index.json and steps backwards 
	// through the versions array to find the absolute highest sensor image tag that is compatible
	// with the currently executing Hub. If the Hub's API is too old for the absolute latest 
	// sensor release, it gracefully injects the previous compatible tag into the docker-compose YAML.
	t.Run("Perfect Match Resolution (Hub API 2)", func(t *testing.T) {
		// Hub API 2 should select v1.5.0, ignoring v2.0.0
		yamlData, err := svc.GetNodeCompose("dummy", "http://localhost", 2)
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}
		
		if !contains(yamlData, "image: test:1.5.0") {
			t.Errorf("Expected to deploy v1.5.0, got yaml:\n%s", string(yamlData))
		}
	})

	t.Run("Legacy Backward Compat (Hub API 4)", func(t *testing.T) {
		// Hub API 4 should select v2.0.0 because 4 >= 3
		yamlData, err := svc.GetNodeCompose("dummy", "http://localhost", 4)
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}
		
		if !contains(yamlData, "image: test:2.0.0") {
			t.Errorf("Expected to deploy v2.0.0, got yaml:\n%s", string(yamlData))
		}
	})
	
	t.Run("No Compatible Version Found", func(t *testing.T) {
		// Hub API 0 is too old for everything (minimum is 1)
		yamlData, err := svc.GetNodeCompose("dummy", "http://localhost", 0)
		if err != nil {
			t.Fatalf("Expected success (generates empty compose, logs warning), got err: %v", err)
		}
		
		if contains(yamlData, "image: test:") {
			t.Errorf("Expected NO sensor to be deployed, but found one in yaml:\n%s", string(yamlData))
		}
	})

	t.Run("Whitespace Robust Parsing", func(t *testing.T) {
		// Even if min_hub_api has spaces like "  3  ", it should parse cleanly and fail on Hub API 2
		yamlData, err := svc.GetNodeCompose("dummy", "http://localhost", 2)
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}
		// Because min_hub_api=" 3 " for v2.0.0 parses successfully, Hub API 2 will correctly reject it and fallback to v1.5.0
		if !contains(yamlData, "image: test:1.5.0") {
			t.Errorf("Expected fallback to v1.5.0, got yaml:\n%s", string(yamlData))
		}
	})

	t.Run("Network Cache Fallback on 502", func(t *testing.T) {
		// First, do a successful fetch to populate the cache
		_, _ = svc.GetNodeCompose("dummy", "http://localhost", 2)

		// Now break the registry URL so the next fetch fails completely
		store.RegistryURL = "http://localhost:1" // guaranteed connection refused

		// Attempt to fetch again. The network will fail, but the cache should save the day!
		yamlData, err := svc.GetNodeCompose("dummy", "http://localhost", 2)
		if err != nil {
			t.Fatalf("Expected cache fallback success, got error: %v", err)
		}
		
		if !contains(yamlData, "image: test:1.5.0") {
			t.Errorf("Expected cached fallback to v1.5.0, got yaml:\n%s", string(yamlData))
		}
	})
}

func contains(b []byte, s string) bool {
	return strings.Contains(string(b), s)
}
