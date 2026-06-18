package event

import (
	"context"
	"fmt"
	"log"
	"strconv"
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
	GetConfigValue(key string) (string, error)
	EnforceRetention(archiveDays, purgeDays int) error
}

type Broadcaster interface {
	Broadcast(topic string, payload interface{})
}

type SiemService interface {
	QueueEvent(event models.Event)
}

type NotifyService interface {
	Dispatch(event models.Event)
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

	whitelistedSources, err := s.store.GetConfigValue("whitelisted_sources")
	if err == nil && whitelistedSources != "" {
		sources := strings.Split(whitelistedSources, ",")
		for _, source := range sources {
			if strings.TrimSpace(source) == e.Source {
				// Whitelisted source, ignore entirely
				return nil
			}
		}
	}

	detailsStr := "{}"
	if len(e.Details) > 0 {
		detailsStr = string(e.Details)
	}
	e.NodeID = nodeID

	lastInsertID, err := s.store.InsertEvent(e, nowStr, detailsStr)
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
		s.notifyService.Dispatch(*e)
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

func (s *Service) StartRetentionWorker(ctx context.Context) {
	log.Println("[Event] Worker started.")

	// Wake up every hour to check retention
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[Event] Worker stopped.")
			return
		case <-ticker.C:
			archiveStr, _ := s.store.GetConfigValue("auto_archive_days")
			purgeStr, _ := s.store.GetConfigValue("auto_purge_days")

			archiveDays, _ := strconv.Atoi(archiveStr)
			purgeDays, _ := strconv.Atoi(purgeStr)

			if archiveDays > 0 || purgeDays > 0 {
				if err := s.store.EnforceRetention(archiveDays, purgeDays); err != nil {
					log.Printf("[WARNING] Event retention task failed: %v", err)
				}
			}
		}
	}
}
