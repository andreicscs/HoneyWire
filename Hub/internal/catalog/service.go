package catalog

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type RegistryIndex struct {
	Sensors []struct {
		ID       string `json:"id"`
		Latest   string `json:"latest"`
		Versions []struct {
			V         string `json:"v"`
			MinHubAPI string `json:"min_hub_api"`
		} `json:"versions"`
	} `json:"sensors"`
}

type Store interface {
	GetConfigValue(key string) (string, error)
}

type Service struct {
	store      Store
	indexCache *RegistryIndex
	mu         sync.RWMutex
}

func NewService(store Store) *Service {
	return &Service{store: store}
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
	s.indexCache = &idx
	s.mu.Unlock()

	return nil
}

func (s *Service) GetIndex() *RegistryIndex {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.indexCache
}

// GetLatestCompatibleVersion safely calculates the highest image tag for the current Hub API
func (s *Service) GetLatestCompatibleVersion(sensorID string, currentHubAPI int) (string, error) {
	s.mu.RLock()
	idx := s.indexCache
	s.mu.RUnlock()

	// If cache is empty, try to refresh it synchronously
	if idx == nil {
		if err := s.RefreshIndex(); err != nil {
			log.Printf("[WARNING] Registry fetch failed (err: %v), no cache available.", err)
			return "", err
		}
		s.mu.RLock()
		idx = s.indexCache
		s.mu.RUnlock()
	}

	for _, sensor := range idx.Sensors {
		if sensor.ID == sensorID {
			for i := len(sensor.Versions) - 1; i >= 0; i-- {
				if reqAPI, err := strconv.Atoi(strings.TrimSpace(sensor.Versions[i].MinHubAPI)); err == nil {
					if currentHubAPI >= reqAPI {
						return sensor.Versions[i].V, nil
					}
				}
			}
			return "", fmt.Errorf("no compatible version found for sensor %s", sensorID)
		}
	}

	return "", fmt.Errorf("sensor %s not found in catalog", sensorID)
}
