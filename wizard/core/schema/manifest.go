package schema

type SensorManifest struct {
	ID               string        `json:"id"`
	Version          string        `json:"version"`
	SchemaVersion    string        `json:"schema_version"`
	MinHubVersion    string        `json:"min_hub_version"`
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
	ImageRepository string           `json:"image_repository"`
	ImageTag        string           `json:"image_tag"`
	ImageDigest     string           `json:"image_digest,omitempty"`
	NetworkMode     string           `json:"network_mode"`
	User            string           `json:"user,omitempty"`
	CapAdd          []string         `json:"cap_add,omitempty"`
	PortAssignments []PortAssignment `json:"port_assignments,omitempty"`
	VolumeMounts    []VolumeMount    `json:"volume_mounts,omitempty"`
	InitContainers  []InitContainer  `json:"init_containers,omitempty"`
	EnvVars         []ConfigVar      `json:"env_vars,omitempty"`
}

type InitContainer struct {
	Name            string        `json:"name"`
	ImageRepository string        `json:"image_repository"`
	ImageTag        string        `json:"image_tag"`
	ImageDigest     string        `json:"image_digest,omitempty"`
	Command         string        `json:"command"`
	VolumeMounts    []VolumeMount `json:"volume_mounts,omitempty"`
}

type VolumeMount struct {
	Type     string `json:"type"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"read_only"`
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
