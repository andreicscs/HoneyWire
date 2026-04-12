package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type Sensor struct {
	SensorID           string
	AgentVersion       string
	hubContractVersion string
	HubEndpoint        string
	HubKey             string
	TestMode           bool
	client             *http.Client
	stopCh             chan struct{}
}

// NewSensor now returns an error instead of calling log.Fatal, making it testable.
func NewSensor() (*Sensor, error) {
	s := &Sensor{
		SensorID:     getEnv("HW_SENSOR_ID", ""),
		AgentVersion: "1.0.0",
		HubEndpoint:  getEnv("HW_HUB_ENDPOINT", ""),
		HubKey:       getEnv("HW_HUB_KEY", ""),
		TestMode:     getEnv("HW_TEST_MODE", "false") == "true",
		client:       &http.Client{Timeout: 10 * time.Second},
		stopCh:       make(chan struct{}),
	}

	if s.HubEndpoint == "" || s.HubKey == "" || s.SensorID == "" {
		return nil, fmt.Errorf("missing required env vars: HW_HUB_ENDPOINT, HW_HUB_KEY, HW_SENSOR_ID")
	}

	return s, nil
}

// Start initiates the background processes and initial sync.
func (s *Sensor) Start() error {
	if err := s.syncHubVersion(); err != nil {
		return err
	}
	go s.heartbeatLoop()
	return nil
}

// Stop gracefully shuts down the heartbeat goroutine.
func (s *Sensor) Stop() {
	close(s.stopCh)
}

func (s *Sensor) RunTestMode() bool {
	log.Println("[*] Test mode: sending synthetic payload...")
	return s.ReportEvent("info", "test_mode_synthetic_alert", "CI/CD Runner", "Mock Hub",
		map[string]any{"test_message": "Automated CI/CD check."})
}

// syncHubVersion attempts to fetch the version with backoff retries.
func (s *Sensor) syncHubVersion() error {
	backoff := []time.Duration{2 * time.Second, 5 * time.Second, 15 * time.Second}

	for i, wait := range backoff {
		err := s.trySyncVersion()
		if err == nil {
			return nil
		}
		log.Printf("[!] Sync attempt %d failed: %v. Retrying in %s", i+1, err, wait)
		time.Sleep(wait)
	}
	return fmt.Errorf("failed to sync with Hub after %d attempts", len(backoff))
}

func (s *Sensor) trySyncVersion() error {
	req, err := http.NewRequest("GET", s.HubEndpoint+"/api/v1/version", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.HubKey)

	resp, err := s.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("hub returned status %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	s.hubContractVersion = result["version"]
	return nil
}

func (s *Sensor) heartbeatLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	s.sendHeartbeat()

	for {
		select {
		case <-ticker.C:
			s.sendHeartbeat()
		case <-s.stopCh:
			return
		}
	}
}

func (s *Sensor) sendHeartbeat() {
	payload := map[string]any{
		"sensor_id": s.SensorID,
		"details": map[string]string{
			"agent_version":    s.AgentVersion,
			"contract_version": s.hubContractVersion,
		},
	}
	s.postToHub("/api/v1/heartbeat", payload)
}

func (s *Sensor) ReportEvent(severity, trigger, source, target string, details map[string]any) bool {
	payload := map[string]any{
		"contract_version": s.hubContractVersion,
		"sensor_id":        s.SensorID,
		"severity":         severity,
		"event_trigger":    trigger,
		"source":           source,
		"target":           target,
		"details":          details,
	}

	resp, err := s.postToHub("/api/v1/event", payload)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil || resp.StatusCode >= 400 {
		log.Printf("[-] Event report failed: %v", err)
		return false
	}
	return true
}

func (s *Sensor) postToHub(path string, data map[string]any) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", s.HubEndpoint+path, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.HubKey)

	return s.client.Do(req)
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}