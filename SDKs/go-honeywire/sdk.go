package sdk

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	SDKDefaultAgentVersion = "1.0.0"
	HoneyWireSchemaVersion = "1.0"
	HeartbeatInterval      = 30 * time.Second
)

// Sensor represents the HoneyWire SDK instance
type Sensor struct {
	SensorType         string
	HubEndpoint        string
	HubKey             string
	SensorID           string
	TestMode           bool
	AgentVersion       string
	Severity           string
	HubContractVersion string
	HTTPClient         *http.Client
}

// EventPayload enforces the strict JSON contract expected by the Hub
type EventPayload struct {
	ContractVersion string         `json:"contract_version"`
	SensorID        string         `json:"sensor_id"`
	SensorType      string         `json:"sensor_type"`
	EventType       string         `json:"event_type"`
	Severity        string         `json:"severity"`
	Timestamp       string         `json:"timestamp"`
	ActionTaken     string         `json:"action_taken"`
	Source          string         `json:"source"`
	Target          string         `json:"target"`
	Details         map[string]any `json:"details"`
}

// NewSensor initializes the SDK, validates environment variables, and returns a Sensor instance
func NewSensor(sensorType string) *Sensor {
	s := &Sensor{
		SensorType:   sensorType,
		HubEndpoint:  os.Getenv("HW_HUB_ENDPOINT"),
		HubKey:       os.Getenv("HW_HUB_KEY"),
		SensorID:     os.Getenv("HW_SENSOR_ID"),
		TestMode:     strings.ToLower(os.Getenv("HW_TEST_MODE")) == "true",
		AgentVersion: getEnv("HONEYWIRE_VERSION", SDKDefaultAgentVersion),
		Severity:     getEnv("HW_SEVERITY", "4"),
		HTTPClient:   &http.Client{Timeout: 5 * time.Second},
	}

	if s.HubEndpoint == "" || s.HubKey == "" || s.SensorID == "" {
		log.Fatal("[!] FATAL: Missing required environment variables (HW_HUB_ENDPOINT, HW_HUB_KEY, HW_SENSOR_ID).")
	}

	return s
}

// Start kicks off the Hub synchronization and background heartbeat
func (s *Sensor) Start() {
	s.syncHubVersion()

	if s.TestMode {
		s.runTestMode()
	}

	// Start the background heartbeat loop
	go s.heartbeatLoop()
}

// ReportEvent formats and sends the payload to the Hub
func (s *Sensor) ReportEvent(eventType, severity string, details map[string]any, actionTaken, source, target string) bool {
	normSeverity := s.normalizeSeverity(severity)

	if actionTaken == "" {
		actionTaken = "logged"
	}
	if source == "" {
		source = "Unknown"
	}
	if target == "" {
		target = "Unknown"
	}

	payload := EventPayload{
		ContractVersion: HoneyWireSchemaVersion,
		SensorID:        s.SensorID,
		SensorType:      s.SensorType,
		EventType:       eventType,
		Severity:        normSeverity,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		ActionTaken:     actionTaken,
		Source:          source,
		Target:          target,
		Details:         details,
	}

	resp, err := s.postToHub("/api/v1/event", payload)
	if err != nil || resp.StatusCode >= 400 {
		log.Printf("[-] Event report failed: %v", err)
		return false
	}
	defer resp.Body.Close()

	log.Printf("[+] Event sent: %s (Severity: %s)", eventType, normSeverity)
	return true
}

func (s *Sensor) syncHubVersion() {
	log.Printf("[*] Synchronizing with Hub at %s...", s.HubEndpoint)

	req, err := http.NewRequest("GET", s.HubEndpoint+"/api/v1/version", nil)
	if err != nil {
		log.Fatalf("[!] FATAL: Failed to build sync request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.HubKey)

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		log.Fatalf("[!] FATAL: Failed to synchronize with Hub. Details: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
		s.HubContractVersion = result["version"]
	} else {
		s.HubContractVersion = "unknown"
	}

	// Semantic Versioning Check
	hubMajor := strings.Split(s.HubContractVersion, ".")[0]
	agentMajor := strings.Split(s.AgentVersion, ".")[0]

	if hubMajor != agentMajor && hubMajor != "unknown" {
		log.Fatalf("[!] FATAL: Version mismatch. Hub (v%s) vs Agent (v%s)", s.HubContractVersion, s.AgentVersion)
	}

	log.Printf("[+] Synchronized successfully. Operating on contract v%s", s.HubContractVersion)
}

func (s *Sensor) heartbeatLoop() {
	payload := map[string]any{
		"sensor_id":   s.SensorID,
		"sensor_type": s.SensorType,
		"details": map[string]string{
			"agent_version":    s.AgentVersion,
			"contract_version": s.HubContractVersion,
		},
	}

	for {
		resp, err := s.postToHub("/api/v1/heartbeat", payload)
		if err != nil {
			log.Printf("[-] Heartbeat error: %v", err)
		} else if resp.StatusCode >= 400 {
			log.Printf("[-] Heartbeat rejected by Hub: Status %d", resp.StatusCode)
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(HeartbeatInterval)
	}
}

func (s *Sensor) runTestMode() {
	log.Println("🛠️ TEST MODE ACTIVE: Sending synthetic payload...")
	success := s.ReportEvent(
		"test_mode_synthetic_alert",
		"info",
		map[string]any{"test_message": "Automated CI/CD check."},
		"ignored",
		"CI/CD Runner",
		"Mock Hub",
	)

	if success {
		log.Println("✅ Test mode complete. Exiting gracefully.")
		os.Exit(0)
	} else {
		log.Println("❌ Test mode failed to contact Hub.")
		os.Exit(1)
	}
}

func (s *Sensor) postToHub(path string, payload any) (*http.Response, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", s.HubEndpoint+path, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.HubKey)
	req.Header.Set("Content-Type", "application/json")

	return s.HTTPClient.Do(req)
}

func (s *Sensor) normalizeSeverity(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	mapping := map[string]string{"1": "info", "2": "low", "3": "medium", "4": "high", "5": "critical"}

	if val, ok := mapping[raw]; ok {
		return val
	}
	switch raw {
	case "info", "low", "medium", "high", "critical":
		return raw
	default:
		log.Printf("[!] Warning: Invalid severity '%s'. Defaulting to 'info'.", raw)
		return "info"
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}