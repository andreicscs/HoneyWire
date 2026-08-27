package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/honeywire/hub/internal/services/config"
)

type mockStore struct {
	values map[string]string
}

func (m *mockStore) GetConfigValue(key string) (string, error) {
	if m.values == nil {
		return "", nil
	}
	return m.values[key], nil
}

func (m *mockStore) CompleteSetup(adminHash, hubEndpoint string) error { return nil }
func (m *mockStore) GetAllConfig() (map[string]string, error) { return m.values, nil }
func (m *mockStore) UpdateConfigBatch(updates map[string]interface{}) error { return nil }
func (m *mockStore) UpdateConfigValue(key, value string) error {
	if m.values == nil {
		m.values = make(map[string]string)
	}
	m.values[key] = value
	return nil
}
func (m *mockStore) FactoryReset() error { return nil }
func (m *mockStore) FactoryResetDryRun() (map[string]int, error) { return nil, nil }

func TestCheckForUpdatesSemver(t *testing.T) {
	releases := []struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Draft   bool   `json:"draft"`
		PreRel  bool   `json:"prerelease"`
	}{
		{"hub/v1.0.0", "http://github.com/v1.0.0", false, false},
		{"hub/v2.1.1", "http://github.com/hub/v2.1.1", false, false},
		{"hub/v2.1.2-rc1", "http://github.com/v2.1.2-rc1", false, true},
		{"wizard/v2.1.1", "http://github.com/wizard/v2.1.1", false, false},
		{"wizard/v2.0.0", "http://github.com/wizard/v2.0.0", false, false},
	}

	b, _ := json.Marshal(releases)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}))
	defer ts.Close()

	origURL := githubAPIURL
	githubAPIURL = ts.URL
	defer func() { githubAPIURL = origURL }()

	store := &mockStore{values: map[string]string{
		"acknowledged_wizard_release": "v2.0.0",
	}}
	cfgSvc := config.NewService(store, nil, nil, nil, "", "2.0.0")

	svc := &Service{
		configService: cfgSvc,
		stopChan:      make(chan struct{}),
	}
	svc.CheckForUpdates()

	state := svc.GetState()
	if !state.UpdateAvailable {
		t.Errorf("Expected Hub UpdateAvailable to be true")
	}
	if state.LatestVersion != "v2.1.1" {
		t.Errorf("Expected Hub LatestVersion to be v2.1.1, got %s", state.LatestVersion)
	}
	if !state.WizardUpdateAvailable {
		t.Errorf("Expected WizardUpdateAvailable to be true")
	}
	if state.LatestWizardVersion != "v2.1.1" {
		t.Errorf("Expected LatestWizardVersion to be v2.1.1, got %s", state.LatestWizardVersion)
	}

	// Test Acknowledging Wizard Release
	if err := cfgSvc.AcknowledgeWizardRelease("v2.1.1"); err != nil {
		t.Fatalf("Failed to acknowledge wizard release: %v", err)
	}
	svc.CheckForUpdates()

	stateAfterAck := svc.GetState()
	if stateAfterAck.WizardUpdateAvailable {
		t.Errorf("Expected WizardUpdateAvailable to be false after acknowledging v2.1.1")
	}
	if stateAfterAck.AcknowledgedWizardRelease != "v2.1.1" {
		t.Errorf("Expected AcknowledgedWizardRelease to be v2.1.1, got %s", stateAfterAck.AcknowledgedWizardRelease)
	}
}

func TestCheckForUpdates_NoFalsePositiveOnStartup(t *testing.T) {
	currentReleases := []struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Draft   bool   `json:"draft"`
		PreRel  bool   `json:"prerelease"`
	}{
		{"hub/v2.0.4", "http://github.com/hub/v2.0.4", false, false},
		{"wizard/v2.0.1", "http://github.com/wizard/v2.0.1", false, false},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(currentReleases)
	}))
	defer ts.Close()

	origURL := githubAPIURL
	githubAPIURL = ts.URL
	defer func() { githubAPIURL = origURL }()

	// Fresh install: no acknowledgment stored yet, Hub version is 2.0.4
	store := &mockStore{values: map[string]string{
		"acknowledged_wizard_release": "",
	}}
	cfgSvc := config.NewService(store, nil, nil, nil, "", "2.0.4")

	svc := &Service{
		configService: cfgSvc,
		stopChan:      make(chan struct{}),
	}
	svc.CheckForUpdates()

	state := svc.GetState()
	if state.UpdateAvailable {
		t.Errorf("Expected Hub UpdateAvailable to be false on fresh install with latest hub version")
	}
	if state.WizardUpdateAvailable {
		t.Errorf("Expected WizardUpdateAvailable to be false on fresh install (should adopt current v2.0.1 baseline)")
	}
	if state.AcknowledgedWizardRelease != "v2.0.1" {
		t.Errorf("Expected AcknowledgedWizardRelease to be initialized to v2.0.1, got %s", state.AcknowledgedWizardRelease)
	}

	// Now simulate a newer Wizard release appearing on GitHub
	currentReleases = append(currentReleases, struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Draft   bool   `json:"draft"`
		PreRel  bool   `json:"prerelease"`
	}{"wizard/v2.0.2", "http://github.com/wizard/v2.0.2", false, false})

	svc.CheckForUpdates()
	stateAfterNewRelease := svc.GetState()
	if !stateAfterNewRelease.WizardUpdateAvailable {
		t.Errorf("Expected WizardUpdateAvailable to be true after new wizard release v2.0.2 is published")
	}
	if stateAfterNewRelease.LatestWizardVersion != "v2.0.2" {
		t.Errorf("Expected LatestWizardVersion to be v2.0.2, got %s", stateAfterNewRelease.LatestWizardVersion)
	}
}
