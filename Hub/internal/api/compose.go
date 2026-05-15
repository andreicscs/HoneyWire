package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sort"

	"github.com/honeywire/hub/internal/auth"
	"gopkg.in/yaml.v3"
)

// --- INCOMING REQUEST PAYLOADS ---
type DeployableSensor struct {
	SensorID  string                 `json:"sensor_id"`
	EnvValues map[string]string      `json:"env_values"`
	Manifest  map[string]interface{} `json:"manifest"`
}

type ComposeReq struct {
	NodeID      string             `json:"node_id"`
	HubEndpoint string             `json:"hub_endpoint"`
	HubKey      string             `json:"hub_key"`
	Sensors     []DeployableSensor `json:"sensors"`
}

// --- OUTGOING COMPOSE STRUCTS (Forces exact YAML ordering) ---
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
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: s.Name}
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
	
	// Security Sandbox
	User        string         `yaml:"user,omitempty"`
	ReadOnly    bool           `yaml:"read_only,omitempty"`
	CapDrop     []string       `yaml:"cap_drop,omitempty"`
	CapAdd      []string       `yaml:"cap_add,omitempty"`
	SecurityOpt []string       `yaml:"security_opt,omitempty"`
	
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

func (h *Handler) GenerateCompose(w http.ResponseWriter, r *http.Request) {
	var req ComposeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// 1. DUAL AUTHENTICATION CHECK
	isAuthorized := false

	if cookie, err := r.Cookie(auth.CookieName); err == nil {
		if h.SessionStore.IsValid(cookie.Value) {
			isAuthorized = true
		}
	}

	if !isAuthorized {
		isAuthorized = h.validateNodeAuth(r, req.NodeID)
	}

	if !isAuthorized {
		http.Error(w, "Unauthorized: Valid UI Session or Node Key required", http.StatusUnauthorized)
		return
	}

	// 2. COMPOSE GENERATION
	var compose ComposeFile

	for _, s := range req.Sensors {
		deployment, ok := s.Manifest["deployment"].(map[string]interface{})
		if !ok {
			continue
		}

		// A. Process Init Containers FIRST (e.g., permission-fixer)
		if initContainers, ok := deployment["init_containers"].([]interface{}); ok {
			for _, ic := range initContainers {
				init := ic.(map[string]interface{})
				initName := init["name"].(string)
				
				initSvc := &ComposeService{
					Image: init["image"].(string),
				}
				if cmd, ok := init["command"]; ok {
					initSvc.Command = cmd.(string)
				}
				
				if vols, ok := init["volume_mounts"].([]interface{}); ok {
					for _, v := range vols {
						volMap := v.(map[string]interface{})
						source := volMap["source"].(string)
						
						source = strings.ReplaceAll(source, "{{ .TrapPath }}", "${TRAP_PATH}")
						for k, val := range s.EnvValues {
							source = strings.ReplaceAll(source, "${"+k+"}", val)
						}
						
						// Use secure long syntax
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
				compose.Services = append(compose.Services, NamedService{Name: initName, Service: initSvc})
			}
		}

		// B. Process Main Sensor Container SECOND
		containerName := s.SensorID
		if !strings.HasPrefix(containerName, "hw-") {
			containerName = "hw-" + containerName
		}

		svc := &ComposeService{
			Image:         deployment["image"].(string),
			ContainerName: containerName,
			Restart:       "unless-stopped",
			
			// GLOBAL SECURITY BASELINE
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

		if nm, ok := deployment["network_mode"].(string); ok && nm != "" {
			svc.NetworkMode = nm
		}
		if user, ok := deployment["user"].(string); ok && user != "" {
			svc.User = user
		}
		if capAdd, ok := deployment["cap_add"].([]interface{}); ok {
			for _, c := range capAdd {
				svc.CapAdd = append(svc.CapAdd, c.(string))
			}
		}

		// Dependencies
		if initContainers, ok := deployment["init_containers"].([]interface{}); ok && len(initContainers) > 0 {
			svc.DependsOn = make(map[string]DependsOn)
			for _, ic := range initContainers {
				initName := ic.(map[string]interface{})["name"].(string)
				svc.DependsOn[initName] = DependsOn{Condition: "service_completed_successfully"}
			}
		}

		// C. Deduplicate Environment Variables
		envMap := make(map[string]string)
		
		// Load manifest defaults first
		if envVars, ok := deployment["env_vars"].([]interface{}); ok {
			for _, e := range envVars {
				eMap := e.(map[string]interface{})
				name := eMap["name"].(string)
				if def, ok := eMap["default"].(string); ok {
					envMap[name] = def
				}
			}
		}
		
		// Load UI user overrides
		for k, v := range s.EnvValues {
			if v != "" {
				envMap[k] = v
			}
		}

		// Force Hub-critical variables
		envMap["HW_HUB_ENDPOINT"] = req.HubEndpoint
		envMap["HW_HUB_KEY"] = req.HubKey
		envMap["HW_NODE_ID"] = req.NodeID
		envMap["HW_SENSOR_ID"] = s.SensorID
		envMap["HW_TEST_MODE"] = "false"

		// --- NEW SORTING LOGIC ---
		// Define the exact order you want the core variables to appear
		coreOrder := []string{
			"HW_HUB_ENDPOINT",
			"HW_HUB_KEY",
			"HW_NODE_ID",
			"HW_SENSOR_ID",
			"HW_TEST_MODE",
			"HW_SEVERITY",
		}

		var envKeys []string
		for k := range envMap {
			envKeys = append(envKeys, k)
		}

		// Sort keys: Core variables first (in defined order), then the rest alphabetically
		sort.Slice(envKeys, func(i, j int) bool {
			k1, k2 := envKeys[i], envKeys[j]
			idx1, idx2 := -1, -1
			
			for idx, val := range coreOrder {
				if k1 == val { idx1 = idx }
				if k2 == val { idx2 = idx }
			}

			if idx1 != -1 && idx2 != -1 { return idx1 < idx2 } // Both are core, sort by coreOrder
			if idx1 != -1 { return true }                      // k1 is core, it goes first
			if idx2 != -1 { return false }                     // k2 is core, it goes first
			return k1 < k2                                     // Neither are core, sort alphabetically
		})

		// Flatten sorted map into slice for YAML
		for _, k := range envKeys {
			svc.Environment = append(svc.Environment, fmt.Sprintf("%s=%s", k, envMap[k]))
		}

		// D. Volumes (Main Container)
		if vols, ok := deployment["volume_mounts"].([]interface{}); ok {
			for _, v := range vols {
				volMap := v.(map[string]interface{})
				source := volMap["source"].(string)
				
				source = strings.ReplaceAll(source, "{{ .TrapPath }}", "${TRAP_PATH}")
				for k, val := range s.EnvValues {
					source = strings.ReplaceAll(source, "${"+k+"}", val)
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

		if svc.NetworkMode != "host" {
			if ports, ok := deployment["port_assignments"].([]interface{}); ok {
				for _, p := range ports {
					portMap := p.(map[string]interface{})
					portNum := int(portMap["default"].(float64))
					svc.Ports = append(svc.Ports, fmt.Sprintf("%d:%d", portNum, portNum))
				}
			}
		}

		compose.Services = append(compose.Services, NamedService{Name: s.SensorID, Service: svc})
	}

	yamlData, _ := yaml.Marshal(&compose)
	
	w.Header().Set("Content-Type", "application/x-yaml")
	w.Write(yamlData)
}