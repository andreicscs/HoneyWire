package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
	Services map[string]*ComposeService `yaml:"services"`
}

type ComposeService struct {
	Image         string                        `yaml:"image"`
	ContainerName string                        `yaml:"container_name,omitempty"`
	Command       string                        `yaml:"command,omitempty"`
	Restart       string                        `yaml:"restart,omitempty"`
	NetworkMode   string                        `yaml:"network_mode,omitempty"`
	User          string                        `yaml:"user,omitempty"`
	DependsOn     map[string]DependsOnCondition `yaml:"depends_on,omitempty"`
	
	// Security Baseline
	ReadOnly    bool           `yaml:"read_only,omitempty"`
	CapDrop     []string       `yaml:"cap_drop,omitempty"`
	CapAdd      []string       `yaml:"cap_add,omitempty"`
	SecurityOpt []string       `yaml:"security_opt,omitempty"`
	Logging     *LoggingConfig `yaml:"logging,omitempty"`
	
	Environment []string `yaml:"environment,omitempty"`
	Ports       []string `yaml:"ports,omitempty"`
	Volumes     []string `yaml:"volumes,omitempty"`
}

type DependsOnCondition struct {
	Condition string `yaml:"condition"`
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
	compose := ComposeFile{
		Services: make(map[string]*ComposeService),
	}

	for _, s := range req.Sensors {
		deployment, ok := s.Manifest["deployment"].(map[string]interface{})
		if !ok {
			continue
		}

		// A. Process Init Containers (e.g., permission-fixer)
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
						
						// Replace ugly Go templates with clean UI variables if missing
						source = strings.ReplaceAll(source, "{{ .TrapPath }}", "${TRAP_PATH}")
						for k, val := range s.EnvValues {
							source = strings.ReplaceAll(source, "${"+k+"}", val)
						}
						
						mount := fmt.Sprintf("%s:%s", source, volMap["target"])
						if ro, ok := volMap["read_only"].(bool); ok && ro {
							mount += ":ro"
						}
						initSvc.Volumes = append(initSvc.Volumes, mount)
					}
				}
				compose.Services[initName] = initSvc
			}
		}

		// B. Process Main Sensor Container
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
			svc.DependsOn = make(map[string]DependsOnCondition)
			for _, ic := range initContainers {
				initName := ic.(map[string]interface{})["name"].(string)
				svc.DependsOn[initName] = DependsOnCondition{Condition: "service_completed_successfully"}
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

		// Force Hub-critical variables (Overwrites placeholders like __HUB_ENDPOINT__)
		envMap["HW_HUB_ENDPOINT"] = req.HubEndpoint
		envMap["HW_HUB_KEY"] = req.HubKey
		envMap["HW_NODE_ID"] = req.NodeID
		envMap["HW_SENSOR_ID"] = s.SensorID
		envMap["HW_TEST_MODE"] = "false"

		// Flatten map into slice for YAML
		for k, v := range envMap {
			svc.Environment = append(svc.Environment, fmt.Sprintf("%s=%s", k, v))
		}

		// D. Volumes
		if vols, ok := deployment["volume_mounts"].([]interface{}); ok {
			for _, v := range vols {
				volMap := v.(map[string]interface{})
				source := volMap["source"].(string)
				
				source = strings.ReplaceAll(source, "{{ .TrapPath }}", "${TRAP_PATH}")
				for k, val := range s.EnvValues {
					source = strings.ReplaceAll(source, "${"+k+"}", val)
				}
				
				mount := fmt.Sprintf("%s:%s", source, volMap["target"])
				if ro, ok := volMap["read_only"].(bool); ok && ro {
					mount += ":ro"
				}
				svc.Volumes = append(svc.Volumes, mount)
			}
		}

		// E. Ports (Only if not host mode)
		if svc.NetworkMode != "host" {
			if ports, ok := deployment["port_assignments"].([]interface{}); ok {
				for _, p := range ports {
					portMap := p.(map[string]interface{})
					portNum := int(portMap["default"].(float64))
					svc.Ports = append(svc.Ports, fmt.Sprintf("%d:%d", portNum, portNum))
				}
			}
		}

		compose.Services[s.SensorID] = svc
	}

	yamlData, _ := yaml.Marshal(&compose)
	
	w.Header().Set("Content-Type", "application/x-yaml")
	w.Write(yamlData)
}