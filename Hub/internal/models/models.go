package models

// SetupPayload represents the initial setup POST request
type SetupPayload struct {
	Password    string `json:"password"`
	HubEndpoint string `json:"hub_endpoint"`
	HubKey      string `json:"hub_key"`
}

// ConfigPayload represents the runtime configuration of the Hub
type ConfigPayload struct {
	HubEndpoint     string   `json:"hub_endpoint"`
	HubKey          string   `json:"hub_key"`
	AutoArchiveDays int      `json:"auto_archive_days"`
	AutoPurgeDays   int      `json:"auto_purge_days"`
	WebhookURL      string   `json:"webhook_url"`
	WebhookType     string   `json:"webhook_type"`
	WebhookEvents   []string `json:"webhook_events"`
	SiemAddress     string   `json:"siem_address"`
	SiemProtocol    string   `json:"siem_protocol"`
}

// Event represents an incoming alert from a sensor
type Event struct {
	ID              int                    `json:"id,omitempty"`
	NodeID          string                 `json:"node_id,omitempty"`
	SensorID        string                 `json:"sensor_id"` // Catalog ID (e.g. hw-tcp-tarpit)`
	Timestamp       string                 `json:"timestamp"`
	ContractVersion string                 `json:"contract_version"`
	EventTrigger    string                 `json:"event_trigger"`
	Severity        string                 `json:"severity"`
	Source          string                 `json:"source"`
	Target          string                 `json:"target"`
	Details         map[string]interface{} `json:"details"`
	IsRead          bool                   `json:"is_read"`
	IsArchived      bool                   `json:"is_archived"`
	Count           int                    `json:"count"`
}

// Heartbeat represents a routine ping from a sensor
type Heartbeat struct {
	SensorID string                 `json:"sensor_id"`
	Metadata map[string]interface{} `json:"metadata"` // Contains HW_CONFIG_REV
}

// Node represents a physical server managing sensors
type Node struct {
	ID               string       `json:"id"`
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
	ID            string                 `json:"id"`
	NodeID        string                 `json:"node_id"`
	Name          string                 `json:"name"`
	Display       string                 `json:"display"`
	Status        string                 `json:"status"`
	LastHeartbeat *string                `json:"lastHeartbeat"`
	IsSilenced    bool                   `json:"isSilenced"`
	Events24h     int                    `json:"events24h"`
	EnvVars       map[string]interface{} `json:"envVars"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// SystemState represents global hub settings
type SystemState struct {
	IsArmed bool `json:"is_armed"`
}
