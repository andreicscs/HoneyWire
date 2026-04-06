package models


// TODO check for fields that are not needed and remove them.
// Event represents an incoming alert from a sensor
type Event struct { 
	ID              int                    `json:"id"`
	Timestamp       string                 `json:"timestamp"`
	ContractVersion string                 `json:"contract_version"`
	SensorID        string                 `json:"sensor_id"`
	SensorType      string                 `json:"sensor_type"`
	EventType       string                 `json:"event_type"`
	Severity        string                 `json:"severity"`
	Source          string                 `json:"source"`
	Target          string                 `json:"target"`
	ActionTaken     string                 `json:"action_taken"`
	Details         map[string]interface{} `json:"details"` // Replaces Python's dict
	IsRead          bool                   `json:"is_read"`
	IsArchived      bool                   `json:"is_archived"`
}

// Heartbeat represents a routine ping from a sensor
type Heartbeat struct {
	SensorID   string                 `json:"sensor_id"`
	SensorType string                 `json:"sensor_type"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Sensor represents a known node in the fleet
type Sensor struct {
	SensorID   string                 `json:"sensor_id"`
	FirstSeen  string                 `json:"first_seen"`
	LastSeen   string                 `json:"last_seen"`
	SensorType string                 `json:"sensor_type"`
	Metadata   map[string]interface{} `json:"metadata"`
	IsSilenced bool                   `json:"is_silenced"`
	Status     string                 `json:"status"`
}

// SystemState represents global hub settings
type SystemState struct {
	IsArmed bool `json:"is_armed"`
}