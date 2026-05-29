package node

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/honeywire/hub/internal/models"
)

// Store defines exactly what the Node service needs from internal/store
type Store interface {
	CreateNode(alias, tags string) (string, string, error)
	UpdateNodeMeta(nodeID, alias, tags string, publicIP, privateIP *string) error
	GetNodes() ([]models.Node, error)
	GetNodeDetails(nodeID string) (*models.Node, error)
	AddSensorToNode(nodeID, sensorID, customName string, configValues map[string]interface{}) error
	UpdateNodeSensor(nodeID, sensorID, customName string, configValues map[string]interface{}) error
	RemoveNodeSensor(nodeID, sensorID string) error
	ClearNodePendingConfig(nodeID string) error
	DeleteNode(nodeID string) error
}

// Broadcaster defines how the service sends real-time updates
type Broadcaster interface {
	Broadcast(topic string, payload interface{})
}

type Service struct {
	store       Store
	broadcaster Broadcaster
}

func NewService(store Store, broadcaster Broadcaster) *Service {
	return &Service{
		store:       store,
		broadcaster: broadcaster,
	}
}

func (s *Service) CreateNode(alias string, tags []string) (string, string, error) {
	tagsJSON, _ := json.Marshal(tags)
	if string(tagsJSON) == "null" {
		tagsJSON = []byte("[]")
	}

	nodeID, apiKey, err := s.store.CreateNode(alias, string(tagsJSON))
	if err != nil {
		return "", "", err
	}

	s.broadcaster.Broadcast("NEW_NODE", map[string]string{"nodeId": nodeID})
	return nodeID, apiKey, nil
}

func (s *Service) UpdateNode(nodeID, alias string, tags []string, publicIP, privateIP *string) error {
	tagsJSON, _ := json.Marshal(tags)
	if string(tagsJSON) == "null" {
		tagsJSON = []byte("[]")
	}

	if err := s.store.UpdateNodeMeta(nodeID, alias, string(tagsJSON), publicIP, privateIP); err != nil {
		return err
	}

	s.broadcaster.Broadcast("UPDATE_NODE", map[string]string{"nodeId": nodeID})
	return nil
}

func (s *Service) GetNodes() ([]models.Node, error) {
	return s.store.GetNodes()
}

func (s *Service) GetNodeDetails(nodeID string) (*models.Node, error) {
	return s.store.GetNodeDetails(nodeID)
}

func (s *Service) AddSensor(nodeID, sensorID, customName string, configValues map[string]interface{}) error {
	if err := s.store.AddSensorToNode(nodeID, sensorID, customName, configValues); err != nil {
		return err
	}
	s.evaluateNodeSyncState(nodeID)
	return nil
}

func (s *Service) EditSensor(nodeID, sensorID, customName string, configValues map[string]interface{}) error {
	if err := s.store.UpdateNodeSensor(nodeID, sensorID, customName, configValues); err != nil {
		return err
	}
	s.evaluateNodeSyncState(nodeID)
	return nil
}

func (s *Service) DeleteSensor(nodeID, sensorID string) error {
	if err := s.store.RemoveNodeSensor(nodeID, sensorID); err != nil {
		return err
	}
	s.evaluateNodeSyncState(nodeID)
	return nil
}

func (s *Service) DeleteNode(nodeID string) error {
	if err := s.store.DeleteNode(nodeID); err != nil {
		return err
	}
	s.broadcaster.Broadcast("DELETE_NODE", map[string]string{"nodeId": nodeID})
	return nil
}

func (s *Service) evaluateNodeSyncState(nodeID string) {
	nodeDetails, err := s.store.GetNodeDetails(nodeID)
	if err == nil {
		newHash := GenerateRevisionHash(nodeDetails.InstalledSensors)
		if newHash == nodeDetails.ActiveRevision {
			s.store.ClearNodePendingConfig(nodeID)
			s.broadcaster.Broadcast("NODE_SYNCED", map[string]string{"nodeId": nodeID})
		}
	}
}

func GenerateRevisionHash(sensors []models.NodeSensor) string {
	type sensorConfig struct {
		ID      string
		EnvVars map[string]interface{}
	}
	var configs []sensorConfig
	for _, s := range sensors {
		configs = append(configs, sensorConfig{ID: s.ID, EnvVars: s.EnvVars})
	}
	sort.Slice(configs, func(i, j int) bool { return configs[i].ID < configs[j].ID })
	b, _ := json.Marshal(configs)
	hash := sha256.Sum256(b)
	return "rev_" + hex.EncodeToString(hash[:4])
}
