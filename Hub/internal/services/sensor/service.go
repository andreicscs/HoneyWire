package sensor

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"
)

// Store defines exactly what the Sensor service needs from internal/store
type Store interface {
	ProcessHeartbeat(nodeID, sensorID, configRev, nowStr string) (bool, error)
	InsertHeartbeat(nodeID, sensorID, minuteBucket string) error
	MarkSensorOffline(nodeID, sensorID, offlineTime string) error
	UpdateSensorSilence(nodeID, sensorID string, silenceVal int) error
	GetTransitionedOfflineNodes(offlineThreshold time.Duration, lastCheck time.Time) (map[string]bool, error)
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

	justSynced, err := s.store.ProcessHeartbeat(nodeID, sensorID, configRev, nowStr)
	if err != nil {
		log.Printf("[ERROR] Heartbeat DB update failed for node %s: %v", nodeID, err)
		return err
	}

	minuteBucket := time.Now().UTC().Truncate(time.Minute).Format(time.RFC3339)
	if err := s.store.InsertHeartbeat(nodeID, sensorID, minuteBucket); err != nil {
		if strings.Contains(err.Error(), "FOREIGN KEY") {
			return ErrSensorNotRegistered
		} else {
			log.Printf("[WARNING] Failed to log heartbeat bucket: %v", err)
		}
	}

	// Dynamic typing for WebSocket payloads to match UI expectations
	if justSynced {
		s.broadcaster.Broadcast("NODE_SYNCED", map[string]interface{}{
			"nodeId": nodeID,
		})
	}

	s.broadcaster.Broadcast("SENSOR_HEARTBEAT", map[string]interface{}{
		"nodeId":    nodeID,
		"sensorId":  sensorID,
		"timestamp": nowStr,
	})

	return nil
}

func (s *Service) StartHealthMonitor(ctx context.Context) {
	log.Println("[Sensor] Worker started.")

	tickerPeriod := 30 * time.Second
	ticker := time.NewTicker(tickerPeriod)
	defer ticker.Stop()

	// Offset lastCheck by the ticker period so the first run catches recent drops
	lastCheck := time.Now().UTC().Add(-tickerPeriod)

	for {
		select {
		case <-ctx.Done():
			log.Println("[Sensor] Worker stopped.")
			return
		case t := <-ticker.C:
			offlineThreshold := 60 * time.Second
			updatedNodeIDs, err := s.store.GetTransitionedOfflineNodes(offlineThreshold, lastCheck)

			if err == nil {
				for nodeID := range updatedNodeIDs {
					s.broadcaster.Broadcast("UPDATE_NODE", map[string]interface{}{
						"id":              nodeID,
						"trigger_refresh": true,
					})
				}
			}
			lastCheck = t.UTC()
		}
	}
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
