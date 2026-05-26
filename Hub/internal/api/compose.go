package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/honeywire/hub/internal/compose"
	"github.com/honeywire/hub/internal/compose/security"
	"github.com/honeywire/hub/internal/models"
	"gopkg.in/yaml.v3"
)

// --- INCOMING REQUEST PAYLOADS ---

type DeployableSensor struct {
	SensorID  string                `json:"sensor_id"`
	EnvValues map[string]string     `json:"env_values"`
	Manifest  models.SensorManifest `json:"manifest"`
}

type ComposeReq struct {
	HubEndpoint string             `json:"hub_endpoint"`
	HubKey      string             `json:"hub_key"`
	Sensors     []DeployableSensor `json:"sensors"`
}


// --- OUTGOING COMPOSE STRUCTS ---

type ComposeFile struct {
	Services OrderedServices `yaml:"services"`
}

type OrderedServices []NamedService

type NamedService struct {
	Name    string
	Service *ComposeService
}

func (os OrderedServices) MarshalYAML() (interface{}, error) {
	node := &yaml.Node{Kind: yaml.MappingNode}

	for _, s := range os {
		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: s.Name,
		}

		valNode := &yaml.Node{}
		if err := valNode.Encode(s.Service); err != nil {
			return nil, err
		}

		node.Content = append(node.Content, keyNode, valNode)
	}

	return node, nil
}

type ComposeService struct {
	Image         string               `yaml:"image"`
	ContainerName string               `yaml:"container_name,omitempty"`
	Command       string               `yaml:"command,omitempty"`
	Restart       string               `yaml:"restart,omitempty"`
	NetworkMode   string               `yaml:"network_mode,omitempty"`
	DependsOn     map[string]DependsOn `yaml:"depends_on,omitempty"`

	// Sandbox
	User        string   `yaml:"user,omitempty"`
	ReadOnly    bool     `yaml:"read_only,omitempty"`
	CapDrop     []string `yaml:"cap_drop,omitempty"`
	CapAdd      []string `yaml:"cap_add,omitempty"`
	SecurityOpt []string `yaml:"security_opt,omitempty"`

	Logging     *LoggingConfig  `yaml:"logging,omitempty"`
	Environment []string        `yaml:"environment,omitempty"`
	Ports       []string        `yaml:"ports,omitempty"`
	Volumes     []ComposeVolume `yaml:"volumes,omitempty"`
}

type DependsOn struct {
	Condition string `yaml:"condition"`
}

type ComposeVolume struct {
	Type     string `yaml:"type"`
	Source   string `yaml:"source"`
	Target   string `yaml:"target"`
	ReadOnly bool   `yaml:"read_only,omitempty"`
}

type LoggingConfig struct {
	Driver  string            `yaml:"driver"`
	Options map[string]string `yaml:"options"`
}

// -----------------------------------------------------------------------------
// MANIFEST FETCHING
// -----------------------------------------------------------------------------

func fetchManifestBytes() ([]byte, error) {
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


// -----------------------------------------------------------------------------
// COMPOSE GENERATOR
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
// NODE-SPECIFIC COMPOSE GENERATION
// -----------------------------------------------------------------------------


// GetNodeCompose generates the official docker-compose.yml for a specific agent
// Authentication: Bearer <API_KEY>
func (h *Handler) GetNodeCompose(w http.ResponseWriter, r *http.Request) {

	authHeader := r.Header.Get("Authorization")

	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		RespondError(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	// -------------------------------------------------------------------------
	// AUTHENTICATE NODE
	// -------------------------------------------------------------------------

	nodeID, err := h.Store.GetNodeByKey(token)

	if err != nil || nodeID == "" {
		RespondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// -------------------------------------------------------------------------
	// LOAD NODE CONFIGURATION
	// -------------------------------------------------------------------------

	nodeDetails, err := h.Store.GetNodeDetails(nodeID)

	if err != nil {
		RespondError(w, "Failed to load node configuration", http.StatusInternalServerError)
		return
	}

	// -------------------------------------------------------------------------
	// RESOLVE HUB ENDPOINT
	// -------------------------------------------------------------------------

	hubEndpoint, err := h.Store.GetConfigValue("hub_endpoint")

	if err != nil || hubEndpoint == "" {
		hubEndpoint = "https://" + r.Host
	}

	// -------------------------------------------------------------------------
	// DETERMINE EFFECTIVE REVISION
	// -------------------------------------------------------------------------

	effectiveRevision := nodeDetails.DesiredRevision

	// If a sync was requested but no revision exists yet,
	// allocate one now.

	if effectiveRevision == "" && nodeDetails.HasPendingConfig {

		effectiveRevision = generateRevisionHash(nodeDetails.InstalledSensors)

		if err := h.Store.SetNodeDesiredRevision(nodeID, effectiveRevision); err != nil {
			RespondError(w, "Failed to allocate node revision", http.StatusInternalServerError)
			return
		}
	}

	// Fallback to active revision

	if effectiveRevision == "" {
		effectiveRevision = nodeDetails.ActiveRevision
	}

	// Final bootstrap fallback

	if effectiveRevision == "" {
		effectiveRevision = generateRevisionHash(nodeDetails.InstalledSensors)
	}
	manifests, fetchErr := fetchStrictCatalogManifests()
	if fetchErr != nil {
		log.Printf("[ERROR] fetchStrictCatalogManifests failed: %v", fetchErr)
		RespondError(w, "Failed to fetch strict manifests", http.StatusInternalServerError)
		return
	}
	manifestByID := make(map[string]models.SensorManifest)
	for _, m := range manifests {
		manifestByID[m.ID] = m
	}

	var finalCompose compose.ComposeFile

	for _, sensor := range nodeDetails.InstalledSensors {
		manifest, ok := manifestByID[sensor.ID]
		if !ok {
			log.Printf("[WARNING] Manifest for sensor %s not found in catalog, skipping.", sensor.ID)
			continue
		}

		// Validate
		if valErr := security.ValidateManifest(manifest); valErr != nil {
			RespondError(w, valErr.Error(), http.StatusBadRequest)
			return
		}

		// Build Env Map
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

		envMap := compose.BuildEnv(manifest, userVars, sysVars)

		// Build service
		svcCompose := compose.BuildService(sensor.ID, manifest, envMap)
		finalCompose.Services = append(finalCompose.Services, svcCompose.Services...)
	}

	yamlData, err := yaml.Marshal(&finalCompose)

	if err != nil {
		RespondError(w, "Failed to generate compose", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.WriteHeader(http.StatusOK)
	w.Write(yamlData)
}

// -----------------------------------------------------------------------------
// PREVIEW COMPOSE API
// -----------------------------------------------------------------------------

func (h *Handler) GenerateCompose(w http.ResponseWriter, r *http.Request) {

	var err error
	var req ComposeReq

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "Invalid payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	var finalCompose compose.ComposeFile

	for _, s := range req.Sensors {
		if valErr := security.ValidateManifest(s.Manifest); valErr != nil {
			http.Error(w, valErr.Error(), http.StatusBadRequest)
			return
		}

		envMap := make(map[string]string)
		for _, v := range s.Manifest.Deployment.EnvVars {
			envMap[v.Name] = v.Default
		}
		for k, v := range s.EnvValues {
			envMap[k] = v
		}
		envMap["HW_HUB_ENDPOINT"] = req.HubEndpoint
		envMap["HW_HUB_KEY"] = req.HubKey
		envMap["HW_SENSOR_ID"] = s.SensorID
		envMap["HW_TEST_MODE"] = "false"

		svcCompose := compose.BuildService(s.SensorID, s.Manifest, envMap)
		finalCompose.Services = append(finalCompose.Services, svcCompose.Services...)
	}
	
	yamlData, err := yaml.Marshal(&finalCompose)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Write(yamlData)
}

func fetchStrictCatalogManifests() ([]models.SensorManifest, error) {
	body, err := fetchManifestBytes()
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
