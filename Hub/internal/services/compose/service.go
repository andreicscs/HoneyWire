package composesvc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	composeEngine "github.com/honeywire/hub/internal/compose"
	"github.com/honeywire/hub/internal/compose/security"
	"github.com/honeywire/hub/internal/models"
	"github.com/honeywire/hub/internal/services/node"
	"gopkg.in/yaml.v3"
)

type Store interface {
	GetNodeByKey(token string) (string, error)
	GetNodeDetails(nodeID string) (*models.Node, error)
	GetConfigValue(key string) (string, error)
	SetNodeDesiredRevision(nodeID, rev string) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// --- DTOs ---

type DeployableSensor struct {
	SensorID  string                `json:"sensorId"`
	EnvValues map[string]string     `json:"envValues"`
	Manifest  models.SensorManifest `json:"manifest"`
}

type PreviewRequest struct {
	HubEndpoint string             `json:"hubEndpoint"`
	HubKey      string             `json:"hubKey"`
	Sensors     []DeployableSensor `json:"sensors"`
}

// --- MANIFEST FETCHING ---

func (s *Service) FetchManifestBytes() ([]byte, error) {
	manifestURL := os.Getenv("HW_MANIFEST_URL")

	if manifestURL == "" {
		manifestURL = "https://raw.githubusercontent.com/andreicscs/HoneyWire/main/Sensors/official/manifests.json"
	}

	resp, err := http.Get(manifestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest registry returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (s *Service) fetchStrictCatalogManifests() ([]models.SensorManifest, error) {
	body, err := s.FetchManifestBytes()
	if err != nil {
		return nil, err
	}

	var rawManifests []json.RawMessage
	if err := json.Unmarshal(body, &rawManifests); err != nil {
		return nil, err
	}

	var manifests []models.SensorManifest
	for _, raw := range rawManifests {
		manifest, err := security.DecodeManifestStrict(bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, manifest)
	}
	return manifests, nil
}

// --- GENERATION LOGIC ---

func (s *Service) GetNodeCompose(token, hostFallback string) ([]byte, error) {
	nodeID, err := s.store.GetNodeByKey(token)
	if err != nil || nodeID == "" {
		return nil, fmt.Errorf("unauthorized")
	}

	nodeDetails, err := s.store.GetNodeDetails(nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed_to_load")
	}

	hubEndpoint, err := s.store.GetConfigValue("hub_endpoint")
	if err != nil || hubEndpoint == "" {
		hubEndpoint = hostFallback
	}

	effectiveRevision := nodeDetails.DesiredRevision
	if effectiveRevision == "" && nodeDetails.HasPendingConfig {
		effectiveRevision = node.GenerateRevisionHash(nodeDetails.InstalledSensors)
		if err := s.store.SetNodeDesiredRevision(nodeID, effectiveRevision); err != nil {
			return nil, fmt.Errorf("failed_to_allocate")
		}
	}
	if effectiveRevision == "" {
		effectiveRevision = nodeDetails.ActiveRevision
	}
	if effectiveRevision == "" {
		effectiveRevision = node.GenerateRevisionHash(nodeDetails.InstalledSensors)
	}

	manifests, fetchErr := s.fetchStrictCatalogManifests()
	if fetchErr != nil {
		log.Printf("[ERROR] fetchStrictCatalogManifests failed: %v", fetchErr)
		return nil, fmt.Errorf("failed_to_fetch")
	}

	manifestByID := make(map[string]models.SensorManifest)
	for _, m := range manifests {
		manifestByID[m.ID] = m
	}

	var finalCompose composeEngine.ComposeFile

	for _, sensor := range nodeDetails.InstalledSensors {
		manifest, ok := manifestByID[sensor.ID]
		if !ok {
			log.Printf("[WARNING] Manifest for sensor %s not found in catalog, skipping.", sensor.ID)
			continue
		}

		if valErr := security.ValidateManifest(manifest); valErr != nil {
			return nil, fmt.Errorf("invalid_manifest: %w", valErr)
		}

		userVars := make(map[string]string)
		for k, v := range sensor.EnvVars {
			if str, ok := v.(string); ok {
				userVars[k] = str
			}
		}

		sysVars := map[string]string{
			"HW_HUB_ENDPOINT": hubEndpoint,
			"HW_HUB_KEY":      token,
			"HW_SENSOR_ID":    sensor.ID,
			"HW_TEST_MODE":    "false",
		}
		if effectiveRevision != "" {
			sysVars["HW_CONFIG_REV"] = effectiveRevision
		}

		envMap := composeEngine.BuildEnv(manifest, userVars, sysVars)
		svcCompose := composeEngine.BuildService(sensor.ID, manifest, envMap)
		finalCompose.Services = append(finalCompose.Services, svcCompose.Services...)
	}

	return yaml.Marshal(&finalCompose)
}

func (s *Service) GeneratePreviewCompose(req PreviewRequest) ([]byte, error) {
	var finalCompose composeEngine.ComposeFile

	for _, sensor := range req.Sensors {
		if valErr := security.ValidateManifest(sensor.Manifest); valErr != nil {
			return nil, fmt.Errorf("invalid_manifest: %w", valErr)
		}

		envMap := make(map[string]string)
		for _, v := range sensor.Manifest.Deployment.EnvVars {
			envMap[v.Name] = v.Default
		}
		for k, v := range sensor.EnvValues {
			envMap[k] = v
		}
		envMap["HW_HUB_ENDPOINT"] = req.HubEndpoint
		envMap["HW_HUB_KEY"] = req.HubKey
		envMap["HW_SENSOR_ID"] = sensor.SensorID
		envMap["HW_TEST_MODE"] = "false"

		svcCompose := composeEngine.BuildService(sensor.SensorID, sensor.Manifest, envMap)
		finalCompose.Services = append(finalCompose.Services, svcCompose.Services...)
	}

	return yaml.Marshal(&finalCompose)
}
