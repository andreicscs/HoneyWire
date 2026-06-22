package composesvc

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/mod/semver"

	"github.com/honeywire/hub/internal/catalog"
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
	SetNodeSensorDeployedVersion(nodeID, sensorID, version string) error
	ApplyNodeRevision(nodeID, revision string) error
}

type Service struct {
	store   Store
	catalog *catalog.Service
	cache   map[string]models.SensorManifest
	mu      sync.RWMutex
}

func NewService(store Store, catalogSvc *catalog.Service) *Service {
	return &Service{
		store:   store,
		catalog: catalogSvc,
		cache:   make(map[string]models.SensorManifest),
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

func (s *Service) FetchManifestBytes(currentHubVersion string) ([]byte, error) {
	manifests, err := s.fetchStrictCatalogManifests(currentHubVersion)
	if err != nil {
		return nil, err
	}
	return json.Marshal(manifests)
}

func (s *Service) fetchStrictCatalogManifests(currentHubVersion string) ([]models.SensorManifest, error) {
	registryURL, err := s.store.GetConfigValue("registry_url")
	if err != nil || registryURL == "" {
		return nil, fmt.Errorf("registry_url is not configured in database")
	}

	if err := s.catalog.RefreshIndex(); err != nil {
		// Suppressed log spam when registry is down
	}

	index := s.catalog.GetIndex()
	if index == nil {
		return nil, fmt.Errorf("registry unreachable and no local index cache available")
	}

	var result []models.SensorManifest

	for _, sensor := range index.Sensors {
		targetVersion, err := s.catalog.GetLatestCompatibleVersion(sensor.ID, currentHubVersion)
		if err != nil || targetVersion == "" {
			log.Printf("[WARNING] No compatible version found for sensor %s", sensor.ID)
			continue
		}

		cacheKey := sensor.ID + "-v" + targetVersion
		s.mu.RLock()
		cached, ok := s.cache[cacheKey]
		s.mu.RUnlock()

		if ok {
			result = append(result, cached)
			continue
		}

		sensorName := strings.TrimPrefix(sensor.ID, "hw-sensor-")
		manifestURL := fmt.Sprintf("%s/%s-v%s.json", strings.TrimRight(registryURL, "/"), sensorName, targetVersion)

		client := &http.Client{Timeout: 10 * time.Second}
		mResp, fetchErr := client.Get(manifestURL)
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

func (s *Service) FetchSpecificManifest(sensorID, targetVersion string) (*models.SensorManifest, error) {
	cleanVersion := strings.TrimPrefix(targetVersion, "v")

	registryURL, err := s.store.GetConfigValue("registry_url")
	if err != nil || registryURL == "" {
		return nil, fmt.Errorf("registry_url is not configured in database")
	}

	sensorName := strings.TrimPrefix(sensorID, "hw-sensor-")
	manifestURL := fmt.Sprintf("%s/%s-v%s.json", strings.TrimRight(registryURL, "/"), sensorName, cleanVersion)

	cacheKey := sensorID + "-v" + cleanVersion
	s.mu.RLock()
	cached, ok := s.cache[cacheKey]
	s.mu.RUnlock()

	if ok {
		return &cached, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	mResp, fetchErr := client.Get(manifestURL)
	if fetchErr != nil {
		log.Printf("[ERROR] FetchSpecificManifest network error: %v", fetchErr)
		return nil, fetchErr
	}
	defer mResp.Body.Close()

	if mResp.StatusCode != 200 {
		log.Printf("[ERROR] FetchSpecificManifest received HTTP %d from %s", mResp.StatusCode, manifestURL)
		return nil, fmt.Errorf("unexpected status %d", mResp.StatusCode)
	}

	var specific models.SensorManifest
	if dErr := json.NewDecoder(mResp.Body).Decode(&specific); dErr != nil {
		log.Printf("[ERROR] FetchSpecificManifest failed to decode JSON: %v", dErr)
		return nil, dErr
	}
	
	s.mu.Lock()
	s.cache[cacheKey] = specific
	s.mu.Unlock()

	return &specific, nil
}

// --- GENERATION LOGIC ---

func (s *Service) GetNodeCompose(token, hostFallback string, currentHubVersion string) ([]byte, error) {
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

	effectiveRevision := node.GenerateRevisionHash(nodeDetails.InstalledSensors, s.catalog, currentHubVersion)
	if effectiveRevision != nodeDetails.DesiredRevision {
		if err := s.store.SetNodeDesiredRevision(nodeID, effectiveRevision); err != nil {
			return nil, fmt.Errorf("failed_to_allocate")
		}
	}

	// Auto-reconcile empty nodes since they have no sensors to report back via heartbeats
	if len(nodeDetails.InstalledSensors) == 0 && nodeDetails.HasPendingConfig {
		_ = s.store.ApplyNodeRevision(nodeID, effectiveRevision)
	}

	var finalCompose composeEngine.ComposeFile

	for _, sensor := range nodeDetails.InstalledSensors {
		targetVersion := sensor.DeployedVersion
		if targetVersion == "" {
			targetVersion, _ = s.catalog.GetLatestCompatibleVersion(sensor.ID, currentHubVersion)
		}

		manifestPtr, err := s.FetchSpecificManifest(sensor.ID, targetVersion)
		if err != nil || manifestPtr == nil {
			log.Printf("[WARNING] Manifest for sensor %s (v%s) not found in catalog, skipping.", sensor.ID, targetVersion)
			continue
		}
		manifest := *manifestPtr

		// The Hub will strictly wait for evaluateNodeSyncState to verify the new hash before considering this version deployed!
		if valErr := security.ValidateManifest(manifest); valErr != nil {
			return nil, fmt.Errorf("invalid_manifest: %w", valErr)
		}

		if manifest.Version != "" {
			reqVer := strings.TrimSpace(manifest.Version)
			if !strings.HasPrefix(reqVer, "v") { reqVer = "v" + reqVer }
			curVer := strings.TrimSpace(currentHubVersion)
			if !strings.HasPrefix(curVer, "v") { curVer = "v" + curVer }
			
			if semver.IsValid(reqVer) && semver.Major(curVer) != semver.Major(reqVer) {
				log.Printf("[ERROR] Sensor %s (v%s) is incompatible with Hub (v%s) - Major versions must match", sensor.ID, reqVer, curVer)
				return nil, fmt.Errorf("incompatible_sensor: %s (v%s) requires a matching Hub Major version", sensor.ID, reqVer)
			}
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
			log.Printf("[ERROR] Compose build failed for sensor %s (Node %s)", sensor.ID, nodeID)
			return nil, fmt.Errorf("build_failed")
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
