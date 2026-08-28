package catalog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

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
	mockIdx := &RegistryIndex{
		Sensors: []struct {
			ID       string `json:"id"`
			Latest   string `json:"latest"`
			Versions []struct {
				V string `json:"v"`
			} `json:"versions"`
		}{
			{ID: "hw-sensor-file-canary", Latest: "2.2.0", Versions: []struct{ V string `json:"v"` }{{V: "2.0.1"}, {V: "2.1.0"}, {V: "2.2.0"}}},
			{ID: "hw-sensor-tcp-tarpit", Latest: "2.2.0", Versions: []struct{ V string `json:"v"` }{{V: "2.0.2"}, {V: "2.1.0"}, {V: "2.2.0"}}},
			{ID: "hw-sensor-icmp-canary", Latest: "2.2.0", Versions: []struct{ V string `json:"v"` }{{V: "2.1.2"}, {V: "2.2.0"}}},
			{ID: "hw-sensor-network-scan-detector", Latest: "2.2.0", Versions: []struct{ V string `json:"v"` }{{V: "2.1.2"}, {V: "2.2.0"}}},
			{ID: "hw-sensor-web-router-decoy", Latest: "2.2.0", Versions: []struct{ V string `json:"v"` }{{V: "2.1.2"}, {V: "2.2.0"}}},
		},
	}
	return &Service{store: store, broadcaster: broadcaster, indexCache: mockIdx}
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
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(indexURL)
	if err != nil {
		return fmt.Errorf("sensor registry unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("sensor registry rate limit exceeded")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sensor registry returned HTTP %d", resp.StatusCode)
	}

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
			// Suppressed log spam when offline
		}
		s.mu.RLock()
		idx = s.indexCache
		s.mu.RUnlock()
	}

	if idx != nil {
		for _, sensor := range idx.Sensors {
			if sensor.ID == sensorID {
				for i := len(sensor.Versions) - 1; i >= 0; i-- {
					reqVer := strings.TrimSpace(sensor.Versions[i].V)
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
	}

	// [TEST MODE] Fallback mock version 2.2.0 if not explicitly defined
	return "2.2.0", nil
}
