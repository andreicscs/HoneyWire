package sensor

import (
	"errors"
	"log"
	"time"
)

// Store defines exactly what the Sensor service needs from internal/store
type Store interface {
	ProcessHeartbeat(nodeID, sensorID, configRev, nowStr string) (bool, error)
	MarkSensorOffline(nodeID, sensorID, offlineTime string) error
	UpdateSensorSilence(nodeID, sensorID string, silenceVal int) error
	GetSensorLastHeartbeat(nodeID, sensorID string) (string, error)
	GetSensorLatestStatus(nodeID, sensorID string) (string, error)
	InsertStatusChange(nodeID, sensorID, status, timestamp string) error
}

// Broadcaster defines how the service sends real-time updates
type Broadcaster interface {
	Broadcast(topic string, payload interface{})
}

type Service struct {
	store       Store
	broadcaster Broadcaster
}

var ErrSensorNotRegistered = errors.New("sensor not registered")

func NewService(store Store, broadcaster Broadcaster) *Service {
	return &Service{
		store:       store,
		broadcaster: broadcaster,
	}
}

// ProcessHeartbeat handles the core logic of a sensor checking in
func (s *Service) ProcessHeartbeat(nodeID, sensorID, configRev string) error {
	nowStr := time.Now().UTC().Format(time.RFC3339)

	latestStatus, err := s.store.GetSensorLatestStatus(nodeID, sensorID)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch latest status for node %s sensor %s: %v", nodeID, sensorID, err)
	}

	// If the recorded state in sensor_status_changes is not "online",
	// this incoming heartbeat marks an online transition.
	isOffline := (latestStatus != "online")

	_, err = s.store.ProcessHeartbeat(nodeID, sensorID, configRev, nowStr)
	if err != nil {
		log.Printf("[ERROR] Heartbeat DB update failed for node %s: %v", nodeID, err)
		return err
	}

	if isOffline {
		if err := s.store.InsertStatusChange(nodeID, sensorID, "online", nowStr); err != nil {
			log.Printf("[WARNING] Failed to log status change: %v", err)
		}
		s.broadcaster.Broadcast("NODE_SYNCED", map[string]interface{}{
			"nodeId": nodeID,
		})
		s.broadcaster.Broadcast("SYNC_CHARTS", nil)
	}

	s.broadcaster.Broadcast("SENSOR_HEARTBEAT", map[string]interface{}{
		"nodeId":    nodeID,
		"sensorId":  sensorID,
		"timestamp": nowStr,
	})

	return nil
}

// ProcessOffline graceful forces a sensor into an offline state
func (s *Service) ProcessOffline(nodeID, sensorID, reason string) error {
	// Push lastHeartbeat 2 hours into the past to instantly force deriveStatus() to return "down"
	offlineTime := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)

	if err := s.store.MarkSensorOffline(nodeID, sensorID, offlineTime); err != nil {
		log.Printf("[ERROR] Failed to set offline status for node %s sensor %s: %v", nodeID, sensorID, err)
		return err
	}

	log.Printf("[INFO] Sensor %s on node %s went offline gracefully (Reason: %s)", sensorID, nodeID, reason)

	s.broadcaster.Broadcast("UPDATE_NODE", map[string]interface{}{
		"nodeId":          nodeID,
		"trigger_refresh": true,
	})

	_ = s.store.InsertStatusChange(nodeID, sensorID, "offline", time.Now().UTC().Format(time.RFC3339))

	return nil
}

// ToggleSilence turns alerting on or off for a specific sensor
func (s *Service) ToggleSilence(nodeID, sensorID string, isSilenced bool) error {
	silenceVal := 0
	if isSilenced {
		silenceVal = 1
	}

	if err := s.store.UpdateSensorSilence(nodeID, sensorID, silenceVal); err != nil {
		return err
	}

	s.broadcaster.Broadcast("SILENCE_SENSOR", map[string]interface{}{
		"nodeId":     nodeID,
		"sensorId":   sensorID,
		"isSilenced": isSilenced,
	})

	return nil
}
