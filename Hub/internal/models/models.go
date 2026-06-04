package models

// SetupPayload represents the initial setup POST request
type SetupPayload struct {
	Password    string `json:"password"`
	HubEndpoint string `json:"hubEndpoint"`
	HubKey      string `json:"hubKey"`
}

// ConfigPayload represents the runtime configuration of the Hub
type ConfigPayload struct {
	HubEndpoint     string   `json:"hubEndpoint"`
	AutoArchiveDays int      `json:"autoArchiveDays"`
	AutoPurgeDays   int      `json:"autoPurgeDays"`
	WebhookURL      string   `json:"webhookUrl"`
	WebhookType     string   `json:"webhookType"`
	WebhookEvents   []string `json:"webhookEvents"`
	SiemAddress     string   `json:"siemAddress"`
	SiemProtocol    string   `json:"siemProtocol"`
}

type Event struct {
	ID              int                    `json:"id,omitempty"`
	NodeID          string                 `json:"nodeId,omitempty"`
	SensorID        string                 `json:"sensorId"`
	Timestamp       string                 `json:"timestamp"`
	ContractVersion string                 `json:"contractVersion"`
	EventTrigger    string                 `json:"eventTrigger"`
	Severity        string                 `json:"severity"`
	Source          string                 `json:"source"`
	Target          string                 `json:"target"`
	Details         map[string]interface{} `json:"details"`
	IsRead          bool                   `json:"isRead"`
	IsArchived      bool                   `json:"isArchived"`
	Count           int                    `json:"count"`
}

// Heartbeat represents a routine ping from a sensor
type Heartbeat struct {
	SensorID string                 `json:"sensorId"`
	Metadata map[string]interface{} `json:"metadata"` // Contains HW_CONFIG_REV
}

// Node represents a physical server managing sensors
type Node struct {
	ID               string       `json:"nodeId"`
	Alias            string       `json:"alias"`
	APIKey           string       `json:"apiKey"`
	ActiveRevision   string       `json:"activeRevision,omitempty"`
	DesiredRevision  string       `json:"desiredRevision,omitempty"`
	PublicIP         *string      `json:"publicIp"`
	PrivateIP        *string      `json:"privateIp"`
	Tags             []string     `json:"tags"`
	HasPendingConfig bool         `json:"hasPendingConfig"`
	LastHeartbeat    *string      `json:"lastHeartbeat"`
	Status           string       `json:"status"` // Derived status (up, down, pending)
	InstalledSensors []NodeSensor `json:"installedSensors"`
}

// NodeSensor represents a deployed sensor on a node
type NodeSensor struct {
	ID            string                 `json:"sensorId"`
	NodeID        string                 `json:"nodeId"`
	Name          string                 `json:"name"`
	Display       string                 `json:"display"`
	Status        string                 `json:"status"`
	LastHeartbeat *string                `json:"lastHeartbeat"`
	IsSilenced    bool                   `json:"isSilenced"`
	Events24h     int                    `json:"events24h"`
	EnvVars       map[string]interface{} `json:"envVars"`
	Metadata      map[string]interface{} `json:"metadata"`
}


// *** manifests ***

type SensorManifest struct {
	ID               string        `json:"id"`
	Version          string        `json:"version"`
	SchemaVersion    string        `json:"schema_version"`
	MinWizardVersion string        `json:"min_wizard_version"`
	Name             string        `json:"name"`
	Category         string        `json:"category"`
	OSILayer         string        `json:"osi_layer"`
	IconSVG          string        `json:"icon_svg"`
	Description      string        `json:"description"`
	Documentation    Documentation `json:"documentation"`
	Heuristics       Heuristics    `json:"heuristics"`
	Deployment       Deployment    `json:"deployment"`
}

type Documentation struct {
	Summary  string       `json:"summary"`
	Sections []DocSection `json:"sections"`
}

type DocSection struct {
	Title   string   `json:"title"`
	Type    string   `json:"type"`
	Content []string `json:"content"`
}

type Heuristics struct {
	Triggers             Triggers `json:"triggers"`
	RecommendationReason string   `json:"recommendation_reason"`
}

type Triggers struct {
	Processes    []string `json:"processes,omitempty"`
	Ports        []int    `json:"ports,omitempty"`
	FilePatterns []string `json:"file_patterns,omitempty"`
	MinMemoryMB  int      `json:"min_memory_mb,omitempty"`
}

type Deployment struct {
	Image           string           `json:"image"`
	NetworkMode     string           `json:"network_mode,omitempty"`
	User            string           `json:"user,omitempty"`
	CapAdd          []string         `json:"cap_add,omitempty"`
	CapDrop         []string         `json:"cap_drop,omitempty"`
	SecurityOpt     []string         `json:"security_opt,omitempty"`
	PortAssignments []PortAssignment `json:"port_assignments,omitempty"`
	VolumeMounts    []VolumeMount    `json:"volume_mounts,omitempty"`
	InitContainers  []InitContainer  `json:"init_containers,omitempty"`
	EnvVars         []ConfigVar      `json:"env_vars,omitempty"`
}

type InitContainer struct {
	Name         string        `json:"name"`
	Image        string        `json:"image"`
	Command      string        `json:"command,omitempty"`
	VolumeMounts []VolumeMount `json:"volume_mounts,omitempty"`
	User         string        `json:"user,omitempty"`
	CapDrop      []string      `json:"cap_drop,omitempty"`
	CapAdd       []string      `json:"cap_add,omitempty"`
	SecurityOpt  []string      `json:"security_opt,omitempty"`
}

type VolumeType string

const (
	DynamicDirBind  VolumeType = "dynamic_dir_bind"
	DynamicFileBind VolumeType = "dynamic_file_bind"
	Bind            VolumeType = "bind"
)

type VolumeMount struct {
	Type         VolumeType `json:"type"`
	Source       string     `json:"source"`
	Target       string     `json:"target"`
	ReadOnly     bool       `json:"read_only"`
	SourceEnv    string     `json:"source_env,omitempty"`
	TargetPrefix string     `json:"target_prefix,omitempty"`
}

type PortAssignment struct {
	EnvVarName string `json:"env_var_name"`
	Default    int    `json:"default"`
}

type ConfigVar struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Default     string `json:"default"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Hidden      bool   `json:"hidden,omitempty"`
}
