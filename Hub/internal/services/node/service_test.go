package node_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/honeywire/hub/internal/catalog"
	"github.com/honeywire/hub/internal/models"
	"github.com/honeywire/hub/internal/services/node"
)

type MockNodeStore struct {
	nodes           map[string]*models.Node
	DeployedUpdates map[string]string // Tracks bumped DeployedVersions for assertion
}

func NewMockNodeStore() *MockNodeStore {
	return &MockNodeStore{
		nodes:           make(map[string]*models.Node),
		DeployedUpdates: make(map[string]string),
	}
}

func (m *MockNodeStore) GetNodes() ([]models.Node, error) {
	var result []models.Node
	for _, n := range m.nodes {
		result = append(result, *n)
	}
	return result, nil
}

func (m *MockNodeStore) GetNodeDetails(nodeID string) (*models.Node, error) {
	if n, ok := m.nodes[nodeID]; ok {
		return n, nil
	}
	return nil, nil
}

func (m *MockNodeStore) SetNodeSensorDeployedVersion(nodeID, sensorID, version string) error {
	m.DeployedUpdates[nodeID+":"+sensorID] = version
	return nil
}

func (m *MockNodeStore) GetConfigValue(key string) (string, error) {
	return "", nil
}

// Unused methods required by interface
func (m *MockNodeStore) CreateNode(alias, tags string) (string, string, error) { return "", "", nil }
func (m *MockNodeStore) UpdateNodeMeta(nodeID, alias, tags string, publicIP, privateIP *string) error { return nil }
func (m *MockNodeStore) AddSensorToNode(nodeID, sensorID, customName string, configValues map[string]interface{}) error { return nil }
func (m *MockNodeStore) UpdateNodeSensor(nodeID, sensorID, customName string, configValues map[string]interface{}) error { return nil }
func (m *MockNodeStore) RemoveNodeSensor(nodeID, sensorID string) error { return nil }
func (m *MockNodeStore) SetNodePendingConfig(nodeID string) error { return nil }
func (m *MockNodeStore) ClearNodePendingConfig(nodeID string) error { return nil }
func (m *MockNodeStore) DeleteNode(nodeID string) error { return nil }

type MockNodeBroadcaster struct{}
func (m *MockNodeBroadcaster) Broadcast(eventType string, payload interface{}) {}

func TestGetNodeDetailsStrictHashMatch(t *testing.T) {
	responseIndex := map[string]interface{}{
		"sensors": []map[string]interface{}{
			{
				"id":     "hw-sensor-test",
				"latest": "v2.0.0", // Latest version available is v2.0.0
				"versions": []map[string]interface{}{
					{"v": "v2.0.0", "min_hub_version": "v1.0.0"},
				},
			},
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(responseIndex)
	}))
	defer ts.Close()

	catStore := &CatalogStore{url: ts.URL}
	catSvc := catalog.NewService(catStore, nil)
	catSvc.RefreshIndex() // Hydrate catalog

	store := NewMockNodeStore()
	store.nodes["node-1"] = &models.Node{
		ID:             "node-1",
		ActiveRevision: "old_revision_hash",
		InstalledSensors: []models.NodeSensor{
			{
				ID:              "hw-sensor-test",
				DeployedVersion: "v1.0.0", // Node currently has v1.0.0
			},
		},
	}

	svc := node.NewService(store, &MockNodeBroadcaster{}, catSvc)

	// Call GetNodeDetails - ActiveRevision ("old_revision_hash") DOES NOT match newly generated hash!
	_, err := svc.GetNodeDetails("node-1")
	if err != nil {
		t.Fatalf("GetNodeDetails failed: %v", err)
	}

	// Because hashes did not match, it MUST NOT auto-bump the DeployedVersion in the database
	if val, ok := store.DeployedUpdates["node-1:hw-sensor-test"]; ok {
		t.Fatalf("Expected DeployedVersion to REMAIN v1.0.0, but it aggressively auto-bumped to %s", val)
	}

	// Now we spoof a successful Edge Node heartbeat that updates ActiveRevision to perfectly match the newest hash
	newHash := node.GenerateRevisionHash(store.nodes["node-1"].InstalledSensors, catSvc, models.HubVersion)
	store.nodes["node-1"].ActiveRevision = newHash

	_, err = svc.GetNodeDetails("node-1")
	if err != nil {
		t.Fatalf("GetNodeDetails failed: %v", err)
	}

	// Because hashes NOW MATCH perfectly, it MUST auto-bump the DeployedVersion
	if val, ok := store.DeployedUpdates["node-1:hw-sensor-test"]; !ok || val != "v2.0.0" {
		t.Fatalf("Expected DeployedVersion to auto-bump to v2.0.0 upon valid hash match, got: %v", val)
	}
}

type CatalogStore struct{ url string }

func (c *CatalogStore) GetConfigValue(key string) (string, error) {
	if key == "registry_url" {
		return c.url, nil
	}
	return "", nil
}
