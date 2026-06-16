package node

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"sort"
	"time"

	"github.com/honeywire/hub/internal/catalog"
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
	SetNodePendingConfig(nodeID string) error
	ClearNodePendingConfig(nodeID string) error
	SetNodeSensorDeployedVersion(nodeID, sensorID, version string) error
	DeleteNode(nodeID string) error
}

// Broadcaster defines how the service sends real-time updates
type Broadcaster interface {
	Broadcast(topic string, payload interface{})
}

type Service struct {
	store       Store
	broadcaster Broadcaster
	catalog     *catalog.Service
}

func NewService(store Store, broadcaster Broadcaster, cat *catalog.Service) *Service {
	return &Service{
		store:       store,
		broadcaster: broadcaster,
		catalog:     cat,
	}
}

// StartWorker runs a background thread that periodically refreshes the catalog
// and recalculates the node sync states to instantly flag updates natively.
func (s *Service) StartWorker(ctx context.Context) {
	log.Println("[INFO] Starting node sync background worker...")

	if s.catalog != nil {
		s.catalog.SetOnChangeHook(func() {
			nodes, err := s.store.GetNodes()
			if err == nil {
				for _, n := range nodes {
					s.evaluateNodeSyncState(n.ID)
				}
			}
		})
	}

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[INFO] Node sync worker stopped")
			return
		case <-ticker.C:
			if s.catalog != nil {
				s.catalog.RefreshIndex()
			}
			nodes, err := s.store.GetNodes()
			if err == nil {
				for _, n := range nodes {
					s.evaluateNodeSyncState(n.ID)
				}
			}
		}
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
	nodes, err := s.store.GetNodes()
	if err != nil {
		return nil, err
	}

	if s.catalog != nil {
		for i := range nodes {
			for _, sensor := range nodes[i].InstalledSensors {
				latest, err := s.catalog.GetLatestCompatibleVersion(sensor.ID, models.HubVersion) 
				if err == nil && latest != "" {
					deployedVer := sensor.DeployedVersion
					if deployedVer != latest {
						newHash := GenerateRevisionHash(nodes[i].InstalledSensors, s.catalog, models.HubVersion)
						if nodes[i].ActiveRevision != "" && nodes[i].ActiveRevision == newHash {
							_ = s.store.SetNodeSensorDeployedVersion(nodes[i].ID, sensor.ID, latest)
							continue
						}
						nodes[i].HasUpdateAvailable = true
						break
					}
				}
			}
		}
	}

	return nodes, nil
}

func (s *Service) GetNodeDetails(nodeID string) (*models.Node, error) {
	node, err := s.store.GetNodeDetails(nodeID)
	if err != nil {
		return nil, err
	}

	// Compute UpdateAvailable for each sensor
	if s.catalog != nil {
		for i, sensor := range node.InstalledSensors {
			latest, err := s.catalog.GetLatestCompatibleVersion(sensor.ID, models.HubVersion)
			if err == nil && latest != "" {
				deployedVer := sensor.DeployedVersion
				if deployedVer != latest {
					newHash := GenerateRevisionHash(node.InstalledSensors, s.catalog, models.HubVersion)
					if node.ActiveRevision != "" && node.ActiveRevision == newHash {
						_ = s.store.SetNodeSensorDeployedVersion(node.ID, sensor.ID, latest)
						node.InstalledSensors[i].DeployedVersion = latest
						continue
					}
					node.InstalledSensors[i].UpdateAvailable = true
					node.HasUpdateAvailable = true
				}
			}
		}
	}

	return node, nil
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
		newHash := GenerateRevisionHash(nodeDetails.InstalledSensors, s.catalog, models.HubVersion)
		if newHash == nodeDetails.ActiveRevision {
			s.store.ClearNodePendingConfig(nodeID)

			if s.catalog != nil {
				for _, sensor := range nodeDetails.InstalledSensors {
					latest, _ := s.catalog.GetLatestCompatibleVersion(sensor.ID, models.HubVersion)
					if latest != "" && sensor.DeployedVersion != latest {
						_ = s.store.SetNodeSensorDeployedVersion(nodeID, sensor.ID, latest)
					}
				}
			}

			s.broadcaster.Broadcast("NODE_SYNCED", map[string]string{"nodeId": nodeID})
		} else if !nodeDetails.HasPendingConfig {
			s.store.SetNodePendingConfig(nodeID)
			s.broadcaster.Broadcast("UPDATE_NODE", map[string]string{"nodeId": nodeID})
		}
	}
}

func GenerateRevisionHash(sensors []models.NodeSensor, catSvc *catalog.Service, currentHubVersion string) string {
	type sensorConfig struct {
		ID      string
		Version string
		EnvVars map[string]interface{}
	}
	var configs []sensorConfig
	for _, s := range sensors {
		latestVersion := ""
		if catSvc != nil {
			latestVersion, _ = catSvc.GetLatestCompatibleVersion(s.ID, currentHubVersion)
		}
		configs = append(configs, sensorConfig{ID: s.ID, Version: latestVersion, EnvVars: s.EnvVars})
	}
	sort.Slice(configs, func(i, j int) bool { return configs[i].ID < configs[j].ID })
	b, _ := json.Marshal(configs)
	hash := sha256.Sum256(b)
	return "rev_" + hex.EncodeToString(hash[:4])
}
