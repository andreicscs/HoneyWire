package updater

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/honeywire/hub/internal/services/config"
	"golang.org/x/mod/semver"
)

var githubAPIURL = "https://api.github.com/repos/AndReicscs/HoneyWire/releases"

type UpdateState struct {
	UpdateAvailable bool   `json:"update_available"`
	LatestVersion   string `json:"latest_version"`
	ReleaseNotesURL string `json:"release_notes_url"`
}

type Service struct {
	configService *config.Service
	mu            sync.RWMutex
	state         UpdateState
	stopChan      chan struct{}
}

func NewService(cfgSvc *config.Service) *Service {
	s := &Service{
		configService: cfgSvc,
		stopChan:      make(chan struct{}),
	}
	s.CheckForUpdates() // Initial check synchronously
	go s.worker()       // Start background loop
	return s
}

func (s *Service) GetState() UpdateState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *Service) Stop() {
	close(s.stopChan)
}

func (s *Service) worker() {
	intervalHours := 24
	cfg, err := s.configService.GetConfig()
	if err == nil {
		intervalHours = cfg.UpdateCheckIntervalHours
	}

	var ticker *time.Ticker
	var tickChan <-chan time.Time
	if intervalHours > 0 {
		ticker = time.NewTicker(time.Duration(intervalHours) * time.Hour)
		tickChan = ticker.C
	}

	// Periodic check for config changes
	configCheckTicker := time.NewTicker(5 * time.Minute)
	defer configCheckTicker.Stop()

	for {
		select {
		case <-tickChan:
			s.CheckForUpdates()
		case <-configCheckTicker.C:
			cfg, err := s.configService.GetConfig()
			if err == nil && cfg.UpdateCheckIntervalHours != intervalHours {
				intervalHours = cfg.UpdateCheckIntervalHours
				if ticker != nil {
					ticker.Stop()
				}
				if intervalHours > 0 {
					ticker = time.NewTicker(time.Duration(intervalHours) * time.Hour)
					tickChan = ticker.C
				} else {
					ticker = nil
					tickChan = nil
				}
			}
		case <-s.stopChan:
			if ticker != nil {
				ticker.Stop()
			}
			return
		}
	}
}

func (s *Service) CheckForUpdates() {
	currV := s.configService.GetVersion()
	if !strings.HasPrefix(currV, "v") {
		currV = "v" + currV
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", githubAPIURL, nil)
	if err != nil {
		log.Printf("[Updater] Failed to create request: %v", err)
		return
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	// If you want to avoid rate limiting as much as possible, a simple user agent is good practice
	req.Header.Set("User-Agent", "HoneyWire-Hub-Updater")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Updater] Failed to fetch releases: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[Updater] GitHub API returned status: %d", resp.StatusCode)
		return
	}

	var releases []struct {
		TagName  string `json:"tag_name"`
		HTMLURL  string `json:"html_url"`
		Draft    bool   `json:"draft"`
		PreRel   bool   `json:"prerelease"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		log.Printf("[Updater] Failed to decode releases: %v", err)
		return
	}

	var highestVersion string = currV
	var latestURL string

	for _, rel := range releases {
		if rel.Draft || rel.PreRel {
			continue
		}
		if strings.HasPrefix(rel.TagName, "hub/v") {
			tagV := strings.TrimPrefix(rel.TagName, "hub/")
			// Ensure valid semver
			if semver.IsValid(tagV) {
				if semver.Compare(tagV, highestVersion) > 0 {
					highestVersion = tagV
					latestURL = rel.HTMLURL
				}
			}
		}
	}

	s.mu.Lock()
	if highestVersion != currV {
		s.state = UpdateState{
			UpdateAvailable: true,
			LatestVersion:   highestVersion,
			ReleaseNotesURL: latestURL,
		}
		log.Printf("[Updater] New version available: %s", highestVersion)
	} else {
		s.state = UpdateState{
			UpdateAvailable: false,
			LatestVersion:   currV,
			ReleaseNotesURL: "",
		}
	}
	s.mu.Unlock()
}
