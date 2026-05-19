package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/honeywire/hub/internal/models"
	"gopkg.in/yaml.v3"
)

// --- INCOMING REQUEST PAYLOADS ---

type DeployableSensor struct {
	SensorID string                 `json:"sensor_id"`
	EnvValues map[string]string     `json:"env_values"`
	Manifest map[string]interface{} `json:"manifest"`
}

type ComposeReq struct {
	HubEndpoint string             `json:"hub_endpoint"`
	HubKey      string             `json:"hub_key"`
	Sensors     []DeployableSensor `json:"sensors"`
}

type composeSensorInput struct {
	SensorID   string
	EnvValues  map[string]string
	Deployment map[string]interface{}
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

	Logging     *LoggingConfig `yaml:"logging,omitempty"`
	Environment []string       `yaml:"environment,omitempty"`
	Ports       []string       `yaml:"ports,omitempty"`
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

func fetchCatalogManifests() ([]map[string]interface{}, error) {
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var manifests []map[string]interface{}

	if err := json.Unmarshal(body, &manifests); err != nil {
		return nil, err
	}

	return manifests, nil
}

// -----------------------------------------------------------------------------
// COMPOSE GENERATOR
// -----------------------------------------------------------------------------

func buildComposeFile(hubEndpoint string, hubKey string, configRev string, sensors []composeSensorInput) ([]byte, error) {

	var compose ComposeFile

	for _, s := range sensors {

		deployment, ok := s.Deployment["deployment"].(map[string]interface{})
		if !ok {
			continue
		}

		// -----------------------------------------------------------------
		// INIT CONTAINERS
		// -----------------------------------------------------------------

		if initContainers, ok := deployment["init_containers"].([]interface{}); ok {

			for _, ic := range initContainers {

				initMap, ok := ic.(map[string]interface{})
				if !ok {
					continue
				}

				initName, _ := initMap["name"].(string)

				initSvc := &ComposeService{
					Image: initMap["image"].(string),
				}

				if cmd, ok := initMap["command"].(string); ok {
					initSvc.Command = cmd
				}

				if vols, ok := initMap["volume_mounts"].([]interface{}); ok {

					for _, v := range vols {

						volMap, ok := v.(map[string]interface{})
						if !ok {
							continue
						}

						source, _ := volMap["source"].(string)

						source = strings.ReplaceAll(
							source,
							"{{ .TrapPath }}",
							"${TRAP_PATH}",
						)

						for k, val := range s.EnvValues {
							source = strings.ReplaceAll(
								source,
								"${"+k+"}",
								val,
							)
						}

						composeVol := ComposeVolume{
							Type:   "bind",
							Source: source,
							Target: volMap["target"].(string),
						}

						if ro, ok := volMap["read_only"].(bool); ok && ro {
							composeVol.ReadOnly = true
						}

						initSvc.Volumes = append(initSvc.Volumes, composeVol)
					}
				}

				compose.Services = append(
					compose.Services,
					NamedService{
						Name:    initName,
						Service: initSvc,
					},
				)
			}
		}

		// -----------------------------------------------------------------
		// MAIN SENSOR SERVICE
		// -----------------------------------------------------------------

		containerName := s.SensorID

		if !strings.HasPrefix(containerName, "hw-") {
			containerName = "hw-" + containerName
		}

		svc := &ComposeService{
			Image:         deployment["image"].(string),
			ContainerName: containerName,

			Restart: "unless-stopped",

			// GLOBAL SANDBOX BASELINE
			ReadOnly:    true,
			CapDrop:     []string{"ALL"},
			SecurityOpt: []string{"no-new-privileges:true"},

			Logging: &LoggingConfig{
				Driver: "json-file",
				Options: map[string]string{
					"max-size": "10m",
					"max-file": "3",
				},
			},
		}

		// -----------------------------------------------------------------
		// OPTIONAL DEPLOYMENT CONFIG
		// -----------------------------------------------------------------

		if nm, ok := deployment["network_mode"].(string); ok && nm != "" {
			svc.NetworkMode = nm
		}

		if user, ok := deployment["user"].(string); ok && user != "" {
			svc.User = user
		}

		if capAdd, ok := deployment["cap_add"].([]interface{}); ok {

			for _, c := range capAdd {

				if capStr, ok := c.(string); ok {
					svc.CapAdd = append(svc.CapAdd, capStr)
				}
			}
		}

		// -----------------------------------------------------------------
		// DEPENDENCIES
		// -----------------------------------------------------------------

		if initContainers, ok := deployment["init_containers"].([]interface{}); ok &&
			len(initContainers) > 0 {

			svc.DependsOn = make(map[string]DependsOn)

			for _, ic := range initContainers {

				initMap, ok := ic.(map[string]interface{})
				if !ok {
					continue
				}

				if initName, ok := initMap["name"].(string); ok {
					svc.DependsOn[initName] = DependsOn{
						Condition: "service_completed_successfully",
					}
				}
			}
		}

		// -----------------------------------------------------------------
		// ENVIRONMENT VARIABLES
		// -----------------------------------------------------------------

		envMap := make(map[string]string)

		// Load manifest defaults first

		if envVars, ok := deployment["env_vars"].([]interface{}); ok {

			for _, e := range envVars {

				eMap, ok := e.(map[string]interface{})
				if !ok {
					continue
				}

				name, _ := eMap["name"].(string)

				if def, ok := eMap["default"].(string); ok {
					envMap[name] = def
				}
			}
		}

		// Block UI override of protected variables

		forbiddenVars := map[string]bool{
			"HW_HUB_ENDPOINT": true,
			"HW_HUB_KEY":      true,
			"HW_SENSOR_ID":    true,
			"HW_CONFIG_REV":   true,
			"HW_TEST_MODE":    true,
		}

		// Apply user overrides

		for k, v := range s.EnvValues {

			if forbiddenVars[k] {
				continue
			}

			if v != "" {
				envMap[k] = v
			}
		}

		// Force hub-owned variables

		envMap["HW_HUB_ENDPOINT"] = hubEndpoint
		envMap["HW_HUB_KEY"] = hubKey
		envMap["HW_SENSOR_ID"] = s.SensorID
		envMap["HW_TEST_MODE"] = "false"

		if configRev != "" {
			envMap["HW_CONFIG_REV"] = configRev
		}

		// Ordered core variables

		coreOrder := []string{
			"HW_HUB_ENDPOINT",
			"HW_HUB_KEY",
			"HW_SENSOR_ID",
			"HW_CONFIG_REV",
			"HW_TEST_MODE",
			"HW_SEVERITY",
		}

		var envKeys []string

		for k := range envMap {
			envKeys = append(envKeys, k)
		}

		sort.Slice(envKeys, func(i, j int) bool {

			k1 := envKeys[i]
			k2 := envKeys[j]

			idx1 := -1
			idx2 := -1

			for idx, val := range coreOrder {

				if k1 == val {
					idx1 = idx
				}

				if k2 == val {
					idx2 = idx
				}
			}

			if idx1 != -1 && idx2 != -1 {
				return idx1 < idx2
			}

			if idx1 != -1 {
				return true
			}

			if idx2 != -1 {
				return false
			}

			return k1 < k2
		})

		for _, k := range envKeys {
			svc.Environment = append(
				svc.Environment,
				fmt.Sprintf("%s=%s", k, envMap[k]),
			)
		}

		// -----------------------------------------------------------------
		// VOLUMES
		// -----------------------------------------------------------------

		if vols, ok := deployment["volume_mounts"].([]interface{}); ok {

			for _, v := range vols {

				volMap, ok := v.(map[string]interface{})
				if !ok {
					continue
				}

				source, _ := volMap["source"].(string)

				source = strings.ReplaceAll(
					source,
					"{{ .TrapPath }}",
					"${TRAP_PATH}",
				)

				for k, val := range s.EnvValues {
					source = strings.ReplaceAll(
						source,
						"${"+k+"}",
						val,
					)
				}

				composeVol := ComposeVolume{
					Type:   "bind",
					Source: source,
					Target: volMap["target"].(string),
				}

				if ro, ok := volMap["read_only"].(bool); ok && ro {
					composeVol.ReadOnly = true
				}

				svc.Volumes = append(svc.Volumes, composeVol)
			}
		}

		// -----------------------------------------------------------------
		// PORTS
		// -----------------------------------------------------------------

		if svc.NetworkMode != "host" {

			if ports, ok := deployment["port_assignments"].([]interface{}); ok {

				for _, p := range ports {

					portMap, ok := p.(map[string]interface{})
					if !ok {
						continue
					}

					if portNum, ok := portMap["default"].(float64); ok {

						svc.Ports = append(
							svc.Ports,
							fmt.Sprintf(
								"%d:%d",
								int(portNum),
								int(portNum),
							),
						)
					}
				}
			}
		}

		compose.Services = append(
			compose.Services,
			NamedService{
				Name:    s.SensorID,
				Service: svc,
			},
		)
	}

	yamlData, err := yaml.Marshal(&compose)
	if err != nil {
		return nil, err
	}

	return yamlData, nil
}

// -----------------------------------------------------------------------------
// NODE-SPECIFIC COMPOSE GENERATION
// -----------------------------------------------------------------------------

func buildComposeFileForNode(hubEndpoint string, hubKey string, configRev string, installedSensors []models.NodeSensor) ([]byte, error) {

	manifests, err := fetchCatalogManifests()
	if err != nil {
		return nil, err
	}

	manifestByID := make(map[string]map[string]interface{})

	for _, manifest := range manifests {

		if id, ok := manifest["id"].(string); ok {
			manifestByID[id] = manifest
		}
	}

	var inputs []composeSensorInput

	for _, sensor := range installedSensors {

		manifest, ok := manifestByID[sensor.ID]
		if !ok {
			continue
		}

		envValues := make(map[string]string)

		for k, v := range sensor.EnvVars {

			if str, ok := v.(string); ok {
				envValues[k] = str
			}
		}

		inputs = append(inputs, composeSensorInput{
			SensorID:   sensor.ID,
			EnvValues:  envValues,
			Deployment: manifest,
		})
	}

	return buildComposeFile(
		hubEndpoint,
		hubKey,
		configRev,
		inputs,
	)
}

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

		effectiveRevision = generateRevisionHash()

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
		effectiveRevision = generateRevisionHash()
	}


	yamlData, err := buildComposeFileForNode(
		hubEndpoint,
		token,
		effectiveRevision,
		nodeDetails.InstalledSensors,
	)

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

	var req ComposeReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	var sensors []composeSensorInput

	for _, s := range req.Sensors {

		sensors = append(sensors, composeSensorInput{
			SensorID:   s.SensorID,
			EnvValues:  s.EnvValues,
			Deployment: s.Manifest,
		})
	}

	yamlData, err := buildComposeFile(
		req.HubEndpoint,
		req.HubKey,
		"",
		sensors,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Write(yamlData)
}