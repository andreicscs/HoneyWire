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
}

// Event represents an incoming alert from a sensor
type Event struct { 
    ID              int                    `json:"id"`
	Timestamp       string                 `json:"timestamp"`
    ContractVersion string                 `json:"contract_version"`
    SensorID        string                 `json:"sensor_id"`
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
	SensorID   string                 `json:"sensor_id"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Sensor represents a known node in the fleet
type Sensor struct {
    SensorID   string                 `json:"sensor_id"`
    FirstSeen  string                 `json:"first_seen"`
    LastSeen   string                 `json:"last_seen"`
    Metadata   map[string]interface{} `json:"metadata"`
    IsSilenced bool                   `json:"is_silenced"`
    Status     string                 `json:"status"`
}

// SystemState represents global hub settings
type SystemState struct {
	IsArmed bool `json:"is_armed"`
}