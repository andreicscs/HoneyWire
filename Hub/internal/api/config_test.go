package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/honeywire/hub/internal/services/config"
	"github.com/honeywire/hub/internal/services/updater"
	"github.com/stretchr/testify/assert"
)

type mockConfigStore struct {
	values map[string]string
}

func (m *mockConfigStore) GetConfigValue(key string) (string, error) {
	if m.values == nil {
		return "", nil
	}
	return m.values[key], nil
}

func (m *mockConfigStore) CompleteSetup(adminHash, hubEndpoint string) error { return nil }
func (m *mockConfigStore) GetAllConfig() (map[string]string, error) { return m.values, nil }
func (m *mockConfigStore) UpdateConfigBatch(updates map[string]interface{}) error { return nil }
func (m *mockConfigStore) UpdateConfigValue(key, value string) error {
	if m.values == nil {
		m.values = make(map[string]string)
	}
	m.values[key] = value
	return nil
}
func (m *mockConfigStore) FactoryReset() error { return nil }
func (m *mockConfigStore) FactoryResetDryRun() (map[string]int, error) { return nil, nil }

type mockCatalog struct {
	err error
}

func (m *mockCatalog) RefreshIndex() error { return m.err }

func TestConfigHandler_CheckForUpdatesManual(t *testing.T) {
	t.Run("Success Response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"tag_name":"hub/v2.1.1","html_url":"https://github.com/test"}]`))
		}))
		defer ts.Close()
		updater.SetGitHubAPIURLForTest(ts.URL)

		store := &mockConfigStore{values: map[string]string{}}
		cfgSvc := config.NewService(store, nil, nil, nil, "", "2.1.0")
		upSvc := updater.NewService(cfgSvc, &mockCatalog{})
		cfg := &config.Config{Env: "test"}
		handler := NewConfigHandler(cfgSvc, cfg, upSvc)

		req := httptest.NewRequest("POST", "/api/v2/system/updates/check", nil)
		rec := httptest.NewRecorder()

		handler.CheckForUpdatesManual(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "latest_version")
	})

	t.Run("Partial Failure Rate Limit (Returns 200 with Warning)", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Ratelimit-Reset", "1724838000")
			w.WriteHeader(http.StatusForbidden)
		}))
		defer ts.Close()
		updater.SetGitHubAPIURLForTest(ts.URL)

		store := &mockConfigStore{values: map[string]string{}}
		cfgSvc := config.NewService(store, nil, nil, nil, "", "2.1.0")
		upSvc := updater.NewService(cfgSvc, &mockCatalog{}) // Catalog succeeds
		cfg := &config.Config{Env: "test"}
		handler := NewConfigHandler(cfgSvc, cfg, upSvc)

		req := httptest.NewRequest("POST", "/api/v2/system/updates/check", nil)
		rec := httptest.NewRecorder()

		handler.CheckForUpdatesManual(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Sensor updates checked. System updates (Hub/Wizard) rate limited (will retry automatically).")
	})

	t.Run("Total Failure Response (Returns 502)", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer ts.Close()
		updater.SetGitHubAPIURLForTest(ts.URL)

		store := &mockConfigStore{values: map[string]string{}}
		cfgSvc := config.NewService(store, nil, nil, nil, "", "2.1.0")
		upSvc := updater.NewService(cfgSvc, &mockCatalog{err: fmt.Errorf("catalog offline")}) // Both fail
		cfg := &config.Config{Env: "test"}
		handler := NewConfigHandler(cfgSvc, cfg, upSvc)

		req := httptest.NewRequest("POST", "/api/v2/system/updates/check", nil)
		rec := httptest.NewRecorder()

		handler.CheckForUpdatesManual(rec, req)

		assert.Equal(t, http.StatusBadGateway, rec.Code)
		assert.Contains(t, rec.Body.String(), "Unable to check updates")
	})
}
