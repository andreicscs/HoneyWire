package catalog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/mod/semver"
)

type RegistryIndex struct {
	Sensors []struct {
		ID       string `json:"id"`
		Latest   string `json:"latest"`
		Versions []struct {
			V string `json:"v"`
		} `json:"versions"`
	} `json:"sensors"`
}

type Store interface {
	GetConfigValue(key string) (string, error)
}

type Broadcaster interface {
	Broadcast(eventType string, payload interface{})
}

type Service struct {
	store       Store
	broadcaster Broadcaster
	indexCache  *RegistryIndex
	mu          sync.RWMutex
	onChange    func()
}

func NewService(store Store, broadcaster Broadcaster) *Service {
	return &Service{store: store, broadcaster: broadcaster}
}

func (s *Service) SetOnChangeHook(hook func()) {
	s.mu.Lock()
	s.onChange = hook
	s.mu.Unlock()
}

func (s *Service) RefreshIndex() error {
	registryURL, err := s.store.GetConfigValue("registry_url")
	if err != nil || registryURL == "" {
		return fmt.Errorf("registry_url not configured")
	}

	indexURL := strings.TrimRight(registryURL, "/") + "/index.json"
	var idx RegistryIndex
	
	resp, err := http.Get(indexURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch registry index")
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&idx); err != nil {
		return err
	}

	s.mu.Lock()
	var oldJSON, newJSON []byte
	if s.indexCache != nil {
		oldJSON, _ = json.Marshal(s.indexCache)
	}
	newJSON, _ = json.Marshal(&idx)
	
	changed := string(oldJSON) != string(newJSON)
	s.indexCache = &idx
	
	var hook func()
	if changed {
		hook = s.onChange
	}
	s.mu.Unlock()

	if changed {
		if s.broadcaster != nil {
			s.broadcaster.Broadcast("CATALOG_UPDATED", nil)
		}
		if hook != nil {
			go hook()
		}
	}

	return nil
}

func (s *Service) GetIndex() *RegistryIndex {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.indexCache
}

// GetLatestCompatibleVersion safely calculates the highest image tag for the current Hub version
func (s *Service) GetLatestCompatibleVersion(sensorID string, currentHubVersion string) (string, error) {
	s.mu.RLock()
	idx := s.indexCache
	s.mu.RUnlock()

	if idx == nil {
		if err := s.RefreshIndex(); err != nil {
			return "", err
		}
		s.mu.RLock()
		idx = s.indexCache
		s.mu.RUnlock()
	}

	if idx == nil {
		return "", fmt.Errorf("registry index not available")
	}

	for _, sensor := range idx.Sensors {
		if sensor.ID == sensorID {
			for i := len(sensor.Versions) - 1; i >= 0; i-- {
				reqVer := strings.TrimSpace(sensor.Versions[i].V)
				// Format semver standard 'vX.Y.Z' for comparison
				if !strings.HasPrefix(reqVer, "v") {
					reqVer = "v" + reqVer
				}
				curVer := strings.TrimSpace(currentHubVersion)
				if !strings.HasPrefix(curVer, "v") {
					curVer = "v" + curVer
				}

				if semver.IsValid(reqVer) && semver.Major(curVer) == semver.Major(reqVer) {
					return sensor.Versions[i].V, nil
				}
			}
			return "", fmt.Errorf("no compatible version found for sensor %s", sensorID)
		}
	}

	return "", fmt.Errorf("sensor %s not found in catalog", sensorID)
}
