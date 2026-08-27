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
	UpdateAvailable           bool   `json:"update_available"`
	LatestVersion             string `json:"latest_version"`
	ReleaseNotesURL           string `json:"release_notes_url"`
	WizardUpdateAvailable     bool   `json:"wizard_update_available"`
	LatestWizardVersion       string `json:"latest_wizard_version"`
	WizardReleaseNotesURL     string `json:"wizard_release_notes_url"`
	AcknowledgedWizardRelease string `json:"acknowledged_wizard_release"`
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

	var highestHubVersion string = currV
	var latestHubURL string
	var highestWizardVersion string
	var latestWizardURL string

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", githubAPIURL, nil)
	if err != nil {
		log.Printf("[Updater] Failed to create request: %v", err)
	} else {
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		// If you want to avoid rate limiting as much as possible, a simple user agent is good practice
		req.Header.Set("User-Agent", "HoneyWire-Hub-Updater")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[Updater] Failed to fetch releases: %v", err)
		} else {
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				var releases []struct {
					TagName string `json:"tag_name"`
					HTMLURL string `json:"html_url"`
					Draft   bool   `json:"draft"`
					PreRel  bool   `json:"prerelease"`
				}

				if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
					log.Printf("[Updater] Failed to decode releases: %v", err)
				} else {
					for _, rel := range releases {
						if rel.Draft || rel.PreRel {
							continue
						}
						if strings.HasPrefix(rel.TagName, "hub/v") {
							tagV := strings.TrimPrefix(rel.TagName, "hub/")
							if !strings.HasPrefix(tagV, "v") {
								tagV = "v" + tagV
							}
							// Ensure valid semver
							if semver.IsValid(tagV) {
								if semver.Compare(tagV, highestHubVersion) > 0 {
									highestHubVersion = tagV
									latestHubURL = rel.HTMLURL
								}
							}
						} else if strings.HasPrefix(rel.TagName, "wizard/v") {
							tagV := strings.TrimPrefix(rel.TagName, "wizard/")
							if !strings.HasPrefix(tagV, "v") {
								tagV = "v" + tagV
							}
							hubMajor := semver.Major(currV)
							if semver.IsValid(tagV) && semver.Major(tagV) == hubMajor {
								if highestWizardVersion == "" || semver.Compare(tagV, highestWizardVersion) > 0 {
									highestWizardVersion = tagV
									latestWizardURL = rel.HTMLURL
								}
							}
						}
					}
				}
			} else {
				log.Printf("[Updater] GitHub API returned status: %d", resp.StatusCode)
			}
		}
	}

	hubMajor := semver.Major(currV)
	ackWizard, _ := s.configService.GetAcknowledgedWizardRelease()
	if ackWizard != "" && !strings.HasPrefix(ackWizard, "v") {
		ackWizard = "v" + ackWizard
	}

	wizardUpdateAvailable := false
	if highestWizardVersion != "" {
		if ackWizard == "" || semver.Major(ackWizard) != hubMajor {
			// On fresh install or major suite upgrade, adopt the latest matching major release as initial baseline
			_ = s.configService.AcknowledgeWizardRelease(highestWizardVersion)
			ackWizard = highestWizardVersion
		} else if semver.Compare(highestWizardVersion, ackWizard) > 0 {
			wizardUpdateAvailable = true
		}
	}

	s.mu.Lock()
	s.state = UpdateState{
		UpdateAvailable:           highestHubVersion != currV,
		LatestVersion:             highestHubVersion,
		ReleaseNotesURL:           latestHubURL,
		WizardUpdateAvailable:     wizardUpdateAvailable,
		LatestWizardVersion:       highestWizardVersion,
		WizardReleaseNotesURL:     latestWizardURL,
		AcknowledgedWizardRelease: ackWizard,
	}
	if highestHubVersion != currV {
		log.Printf("[Updater] New Hub version available: %s", highestHubVersion)
	}
	if wizardUpdateAvailable {
		log.Printf("[Updater] New Wizard version available: %s (acknowledged: '%s')", highestWizardVersion, ackWizard)
	}
	s.mu.Unlock()
}
