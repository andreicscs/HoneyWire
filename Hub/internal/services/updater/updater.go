package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/honeywire/hub/internal/services/config"
	"golang.org/x/mod/semver"
)

var githubAPIURL = "https://api.github.com/repos/AndReicscs/HoneyWire/releases"

// SetGitHubAPIURLForTest overrides GitHub API URL during testing
func SetGitHubAPIURLForTest(url string) {
	githubAPIURL = url
}

type UpdateState struct {
	UpdateAvailable           bool   `json:"update_available"`
	LatestVersion             string `json:"latest_version"`
	ReleaseNotesURL           string `json:"release_notes_url"`
	WizardUpdateAvailable     bool   `json:"wizard_update_available"`
	LatestWizardVersion       string `json:"latest_wizard_version"`
	WizardReleaseNotesURL     string `json:"wizard_release_notes_url"`
	AcknowledgedWizardRelease string `json:"acknowledged_wizard_release"`
	Warning                   string `json:"warning,omitempty"`
}

type CatalogRefresher interface {
	RefreshIndex() error
}

type Service struct {
	configService  *config.Service
	catalog        CatalogRefresher
	mu             sync.RWMutex
	state          UpdateState
	retryTriggerCh chan time.Duration
	stopChan       chan struct{}
}

func NewService(cfgSvc *config.Service, catalog CatalogRefresher) *Service {
	s := &Service{
		configService:  cfgSvc,
		catalog:        catalog,
		retryTriggerCh: make(chan time.Duration, 5),
		stopChan:       make(chan struct{}),
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

func parseBackoffDuration(resp *http.Response) time.Duration {
	if val := resp.Header.Get("X-Ratelimit-Reset"); val != "" {
		if epoch, err := strconv.ParseInt(val, 10, 64); err == nil {
			resetTime := time.Unix(epoch, 0)
			diff := time.Until(resetTime)
			if diff > 0 {
				return diff + 5*time.Second
			}
		}
	}
	if val := resp.Header.Get("Retry-After"); val != "" {
		if secs, err := strconv.Atoi(val); err == nil && secs > 0 {
			return time.Duration(secs)*time.Second + 5*time.Second
		}
	}
	return 1 * time.Hour
}

func (s *Service) worker() {
	log.Println("[Updater] Worker started.")
	defer log.Println("[Updater] Worker stopped.")

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

	var retryTimer *time.Timer
	var retryChan <-chan time.Time

	// Periodic check for config changes
	configCheckTicker := time.NewTicker(5 * time.Minute)
	defer configCheckTicker.Stop()

	for {
		select {
		case <-tickChan:
			s.CheckForUpdates()
		case backoff := <-s.retryTriggerCh:
			if retryTimer != nil {
				retryTimer.Stop()
			}
			if backoff > 0 {
				log.Printf("[Updater] Scheduling automatic retry in %v", backoff.Round(time.Second))
				retryTimer = time.NewTimer(backoff)
				retryChan = retryTimer.C
			} else {
				retryTimer = nil
				retryChan = nil
			}
		case <-retryChan:
			log.Println("[Updater] Executing scheduled retry check after rate limit backoff.")
			retryTimer = nil
			retryChan = nil
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
			if retryTimer != nil {
				retryTimer.Stop()
			}
			return
		}
	}
}

func (s *Service) CheckForUpdates() error {
	var catalogErr error
	var releasesErr error
	var backoffDuration time.Duration

	if s.catalog != nil {
		if err := s.catalog.RefreshIndex(); err != nil {
			log.Printf("[Updater] Failed to refresh sensor catalog: %v", err)
			catalogErr = fmt.Errorf("Sensor catalog refresh failed: %w", err)
		}
	}

	currV := s.configService.GetVersion()
	if !strings.HasPrefix(currV, "v") {
		currV = "v" + currV
	}

	var highestHubVersion string = currV
	var latestHubURL string
	var highestWizardVersion string
	var latestWizardURL string
	var wizardUpdateAvailable bool

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", githubAPIURL, nil)
	if err != nil {
		log.Printf("[Updater] Failed to create request: %v", err)
		releasesErr = fmt.Errorf("Failed to initialize update request: %w", err)
	} else {
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		req.Header.Set("User-Agent", "HoneyWire-Hub-Updater")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[Updater] Failed to fetch releases: %v", err)
			releasesErr = fmt.Errorf("Unable to connect to update servers: %w", err)
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
					releasesErr = fmt.Errorf("Failed to parse release payload: %w", err)
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
				bodyBytes, _ := io.ReadAll(resp.Body)
				log.Printf("[Updater] Update service returned status %d: %s", resp.StatusCode, string(bodyBytes))
				if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
					backoffDuration = parseBackoffDuration(resp)
					releasesErr = fmt.Errorf("Rate limit exceeded. Please try again later.")
				} else {
					releasesErr = fmt.Errorf("Update service returned error (HTTP %d)", resp.StatusCode)
				}
			}
		}
	}

	// Trigger worker automatic retry on any failure
	if (releasesErr != nil || catalogErr != nil) && s.retryTriggerCh != nil {
		if backoffDuration == 0 {
			backoffDuration = 1 * time.Hour
		}
		select {
		case s.retryTriggerCh <- backoffDuration:
		default:
		}
	}


	hubMajor := semver.Major(currV)
	ackWizard, _ := s.configService.GetAcknowledgedWizardRelease()
	if ackWizard != "" && !strings.HasPrefix(ackWizard, "v") {
		ackWizard = "v" + ackWizard
	}

	if highestWizardVersion != "" && !wizardUpdateAvailable {
		if ackWizard == "" || semver.Major(ackWizard) != hubMajor {
			_ = s.configService.AcknowledgeWizardRelease(highestWizardVersion)
			ackWizard = highestWizardVersion
		} else if semver.Compare(highestWizardVersion, ackWizard) > 0 {
			wizardUpdateAvailable = true
		}
	}

	var warning string
	if releasesErr != nil && catalogErr == nil {
		warning = "Sensor updates checked. System updates (Hub/Wizard) rate limited (will retry automatically)."
	} else if catalogErr != nil && releasesErr == nil {
		warning = "System updates (Hub/Wizard) checked. Sensor registry is currently unreachable (will retry automatically)."
	}

	s.mu.Lock()
	if releasesErr == nil {
		s.state = UpdateState{
			UpdateAvailable:           highestHubVersion != currV && highestHubVersion != "",
			LatestVersion:             highestHubVersion,
			ReleaseNotesURL:           latestHubURL,
			WizardUpdateAvailable:     wizardUpdateAvailable,
			LatestWizardVersion:       highestWizardVersion,
			WizardReleaseNotesURL:     latestWizardURL,
			AcknowledgedWizardRelease: ackWizard,
			Warning:                   warning,
		}
	} else {
		s.state.Warning = warning
	}
	s.mu.Unlock()

	// If BOTH failed, return error to API handler (triggers HTTP 502)
	if releasesErr != nil && catalogErr != nil {
		return fmt.Errorf("Unable to check updates: %v; %v (will retry automatically)", releasesErr, catalogErr)
	}

	// Partial failure or complete success: Return nil so API responds with HTTP 200 containing warning
	return nil
}
