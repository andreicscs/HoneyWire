package event

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/honeywire/hub/internal/models"
)

type Store interface {
	InsertEvent(e *models.Event, timestamp string, detailsJSON string) (int, error)
	UpdateNodeLastHeartbeat(nodeID, sensorID, timestamp string) error
	IsSensorSilenced(nodeID, sensorID string) (bool, error)
	GetEvents(isArchived int, nodeID, sensorID string) ([]models.Event, error)
	GetUnreadEventCount() (int, error)
	MarkEventRead(eventID string) error
	MarkAllEventsRead() error
	ArchiveEvent(eventID string) error
	ArchiveAllEvents() error
	GetEventCount() (int, error)
	ClearAllEvents() error
}

type Broadcaster interface {
	Broadcast(topic string, payload interface{})
}

type SiemService interface {
	QueueEvent(event models.Event)
}

type NotifyService interface {
	Dispatch(title, message, severity string)
}

type Service struct {
	store         Store
	broadcaster   Broadcaster
	siemService   SiemService
	notifyService NotifyService
	hubVersion    string
}

func NewService(store Store, broadcaster Broadcaster, siemService SiemService, notifyService NotifyService, hubVersion string) *Service {
	return &Service{
		store:         store,
		broadcaster:   broadcaster,
		siemService:   siemService,
		notifyService: notifyService,
		hubVersion:    hubVersion,
	}
}

// ProcessEvent handles incoming events from sensors, pushes them to the SIEM, and triggers notifications.
func (s *Service) ProcessEvent(e *models.Event, nodeID string) error {
	hubMajor := strings.Split(s.hubVersion, ".")[0]
	agentMajor := strings.Split(e.ContractVersion, ".")[0]
	if agentMajor == "" || hubMajor != agentMajor {
		return fmt.Errorf("upgrade required")
	}

	nowStr := time.Now().UTC().Format(time.RFC3339)
	detailsJSON, _ := json.Marshal(e.Details)
	e.NodeID = nodeID

	lastInsertID, err := s.store.InsertEvent(e, nowStr, string(detailsJSON))
	if err != nil {
		if strings.Contains(err.Error(), "FOREIGN KEY") {
			return fmt.Errorf("sensor_not_registered")
		}
		log.Printf("[ERROR] Failed to insert event for node %s/sensor %s: %v", nodeID, e.SensorID, err)
		return err
	}

	e.ID = lastInsertID
	e.Timestamp = nowStr

	s.store.UpdateNodeLastHeartbeat(nodeID, e.SensorID, nowStr)

	isSilenced, err := s.store.IsSensorSilenced(nodeID, e.SensorID)
	if err != nil {
		log.Printf("[WARNING] Failed to check silence status for node %s/sensor %s: %v", nodeID, e.SensorID, err)
	}

	if !isSilenced {
		title := fmt.Sprintf("Intrusion Alert: %s", e.SensorID)
		message := fmt.Sprintf("Trigger: %s\nSource: %s\nTarget: %s", e.EventTrigger, e.Source, e.Target)
		s.notifyService.Dispatch(title, message, e.Severity)
	}

	s.siemService.QueueEvent(*e)

	s.broadcaster.Broadcast("NEW_EVENT", *e)
	return nil
}

func (s *Service) GetEvents(isArchived int, nodeID, sensorID string) ([]models.Event, error) {
	return s.store.GetEvents(isArchived, nodeID, sensorID)
}

func (s *Service) GetUnreadCount() (int, error) {
	return s.store.GetUnreadEventCount()
}

func (s *Service) MarkSingleEventRead(eventID string) error { return s.store.MarkEventRead(eventID) }
func (s *Service) MarkEventsRead() error                    { return s.store.MarkAllEventsRead() }
func (s *Service) ArchiveEvent(eventID string) error        { return s.store.ArchiveEvent(eventID) }
func (s *Service) ArchiveAll() error                        { return s.store.ArchiveAllEvents() }

func (s *Service) ClearEvents(dryrun bool, ip string) (int, error) {
	if dryrun {
		return s.store.GetEventCount()
	}
	log.Printf("[!] AUDIT: Database purged by IP %s", ip)
	return 0, s.store.ClearAllEvents()
}
