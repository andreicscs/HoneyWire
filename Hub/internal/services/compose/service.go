package composesvc

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"


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
	cache map[string]models.SensorManifest
	mu    sync.RWMutex
}

func NewService(store Store) *Service {
	return &Service{
		store: store,
		cache: make(map[string]models.SensorManifest),
	}
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

// --- MANIFEST FETCHING ---

func (s *Service) FetchManifestBytes() ([]byte, error) {
	manifests, err := s.fetchStrictCatalogManifests()
	if err != nil {
		return nil, err
	}
	return json.Marshal(manifests)
}

func (s *Service) fetchStrictCatalogManifests() ([]models.SensorManifest, error) {
	registryURL, err := s.store.GetConfigValue("registry_url")
	if err != nil || registryURL == "" {
		registryURL = "https://raw.githubusercontent.com/andreicscs/HoneyWire/registry-pages"
	}

	indexURL := strings.TrimRight(registryURL, "/") + "/index.json"
	resp, err := http.Get(indexURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	var idx struct {
		Sensors []struct {
			ID     string `json:"id"`
			Latest string `json:"latest"`
		} `json:"sensors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&idx); err != nil {
		return nil, err
	}

	var result []models.SensorManifest

	for _, sensor := range idx.Sensors {
		cacheKey := sensor.ID + "-v" + sensor.Latest
		s.mu.RLock()
		cached, ok := s.cache[cacheKey]
		s.mu.RUnlock()

		if ok {
			result = append(result, cached)
			continue
		}

		sensorName := strings.TrimPrefix(sensor.ID, "hw-sensor-")
		manifestURL := fmt.Sprintf("%s/%s-v%s.json", strings.TrimRight(registryURL, "/"), sensorName, sensor.Latest)

		mResp, fetchErr := http.Get(manifestURL)
		if fetchErr != nil {
			log.Printf("[WARNING] Failed to fetch %s: %v", manifestURL, fetchErr)
			continue
		}

		if mResp.StatusCode != http.StatusOK {
			mResp.Body.Close()
			continue
		}

		manifest, decodeErr := security.DecodeManifestStrict(mResp.Body)
		mResp.Body.Close()
		if decodeErr != nil {
			log.Printf("[WARNING] Failed to decode %s: %v", manifestURL, decodeErr)
			continue
		}

		s.mu.Lock()
		s.cache[cacheKey] = manifest
		s.mu.Unlock()

		result = append(result, manifest)
	}

	return result, nil
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
		svcCompose, err := composeEngine.BuildService(sensor.ID, manifest, envMap)
		if err != nil {
			log.Printf("[ERROR] Compose build failed for sensor %s (Node %s): %v", sensor.ID, nodeID, err)
			return nil, fmt.Errorf("build_failed: %w", err)
		}
		finalCompose.Services = append(finalCompose.Services, svcCompose.Services...)
	}

	return yaml.Marshal(&finalCompose)
}

func (s *Service) GeneratePreviewCompose(req PreviewRequest) ([]byte, error) {
	var finalCompose composeEngine.ComposeFile

	for _, sensor := range req.Sensors {
		if valErr := security.ValidateManifest(sensor.Manifest); valErr != nil {
			log.Printf("[ERROR] Preview manifest validation failed for sensor %s: %v", sensor.SensorID, valErr)
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

		svcCompose, err := composeEngine.BuildService(sensor.SensorID, sensor.Manifest, envMap)
		if err != nil {
			log.Printf("[ERROR] Compose preview build failed for sensor %s: %v", sensor.SensorID, err)
			return nil, fmt.Errorf("build_failed: %w", err)
		}
		finalCompose.Services = append(finalCompose.Services, svcCompose.Services...)
	}

	return yaml.Marshal(&finalCompose)
}
