package store

import (
	"time"

	"github.com/honeywire/hub/internal/models"
)

// DataStore defines the interface for all database operations.
type DataStore interface {
	GetConfigValue(key string) (string, error)
	UpdateConfigValue(key, value string) error
	GetAllConfig() (map[string]string, error)
	CompleteSetup(adminHash, hubEndpoint, hubKey string) error
	UpdateConfigBatch(req map[string]interface{}) error
	FactoryReset() error

	// Events
	InsertEvent(e *models.Event, nowStr string, detailsStr string) (int, error)
	UpdateNodeLastSeen(nodeID, timestamp string) error
	IsSensorSilenced(nodeID, sensorID string) (bool, error)
	GetEvents(isArchived int, nodeID string, sensorID string) ([]models.Event, error)
	GetUnreadEventCount() (int, error)
	MarkEventRead(eventID string) error
	MarkAllEventsRead() error
	ArchiveEvent(eventID string) error
	ArchiveAllEvents() error
	GetEventCount() (int, error)
	ClearAllEvents() error

	// Sensors
	UpsertSensor(hb *models.Heartbeat, nowStr, metadataStr string) error
	InsertHeartbeat(nodeID, sensorID, timeBucket string) error
	GetAllSensors() ([]models.NodeSensor, error)
	GetSensorsForUptime(nowStr string) ([]SensorUptimeData, error)
	GetHeartbeatsSince(cutoffStr string) ([]HeartbeatData, error)
	UpdateSensorSilence(nodeID, sensorID string, silenceVal int) error
	DeleteSensor(nodeID, sensorID string) (int64, error)
	MarkSensorOffline(nodeID, sensorID, offlineTime string) error

	// Provisioning
	InsertPairingToken(token string, expiresAt time.Time, createdAt time.Time) error
	ValidatePairingToken(token string) (bool, error)
	CreateNode(nodeID, alias, nodeKey, ipAddress, nowStr string) error
	GetNodeKey(nodeID string) (string, error)
}
