package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"

	// codeql[go/insecure-randomness] Non-cryptographic use case.
	// nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const (
	MaxRetriesPerEvent    = 7
	BaseHeartbeatInterval = 30 * time.Second
	TerminalSleepInterval = 1 * time.Hour
)

// ============================================================================
// SHARED CLASSIFIER
// ============================================================================

type ResponseFact struct {
	IsError     bool
	IsTransient bool
	StatusCode  int
	RetryAfter  time.Duration
}

// classify assesses the raw HTTP result and returns factual state, not behavior.
func classify(err error, resp *http.Response) ResponseFact {
	if err != nil {
		return ResponseFact{IsError: true, IsTransient: true} // Network drop/timeout
	}

	fact := ResponseFact{
		StatusCode: resp.StatusCode,
		IsError:    resp.StatusCode >= 400,
	}

	if fact.IsError {
		// Determine if the error is recoverable (Transient) or fatal (Terminal)
		switch resp.StatusCode {
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
			fact.IsTransient = false
		default:
			fact.IsTransient = true // 429, 5xx, etc.

			// Extract explicit wait instructions if the server provides them
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if sec, err := strconv.Atoi(ra); err == nil {
					fact.RetryAfter = time.Duration(sec) * time.Second
				}
			}
		}
	}

	return fact
}

// ============================================================================
// 2. POLICY INTERPRETERS (Domain-Specific Rules)
// ============================================================================

type EventAction string

const (
	EventSuccess EventAction = "success"
	EventRetry   EventAction = "retry"
	EventDrop    EventAction = "drop"
)

// eventPolicy governs the strict, ordered, retry-bounded queue.
func (s *Sensor) eventPolicy(fact ResponseFact, attempt int) (EventAction, time.Duration) {
	if !fact.IsError {
		return EventSuccess, 0
	}
	if !fact.IsTransient {
		return EventDrop, 0
	}

	delay := fact.RetryAfter
	if delay == 0 {
		delay = s.calculateBackoff(attempt)
	}
	return EventRetry, delay
}

// heartbeatPolicy governs the stateless, lossy, continuous signal.
// It doesn't have "actions"—it only dictates the cadence of the next pulse.
func (s *Sensor) heartbeatPolicy(fact ResponseFact) time.Duration {
	if !fact.IsError {
		return BaseHeartbeatInterval // State 1: OK
	}
	if !fact.IsTransient {
		return TerminalSleepInterval // State 3: Terminal Cooldown
	}

	// State 2: Transient/Degraded
	if fact.RetryAfter > BaseHeartbeatInterval {
		return fact.RetryAfter // Hub explicitly asked for breathing room
	}
	return BaseHeartbeatInterval // Default: maintain steady lossy pulse
}

func (s *Sensor) calculateBackoff(attempt int) time.Duration {
	base := 2.0
	maxDelay := 60.0
	delay := base * math.Pow(2, float64(attempt))
	if delay > maxDelay {
		delay = maxDelay
	}

	// codeql[go/insecure-randomness] Non-cryptographic use case.
	// nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
	jitter := (s.rng.Float64() * 0.2) - 0.1
	finalDelay := delay + (delay * jitter)
	return time.Duration(finalDelay * float64(time.Second))
}

// ============================================================================
// SENSOR STRUCT & INIT
// ============================================================================

type Sensor struct {
	SensorID           string
	Severity           string
	AgentVersion       string
	hubContractVersion string
	HubEndpoint        string
	HubKey             string
	ConfigRev          string
	TestMode           bool
	client             *http.Client

	eventCh     chan map[string]any
	stopCh      chan struct{}
	rng         *rand.Rand
	testPayload map[string]any
	testTrigger string
	testSource  string
	testTarget  string
	testDetails map[string]any
}

func NewSensor() (*Sensor, error) {
	s := &Sensor{
		SensorID:     getEnv("HW_SENSOR_ID", ""),
		Severity:     getEnv("HW_SEVERITY", "medium"),
		AgentVersion: "1.0.0",
		HubEndpoint:  getEnv("HW_HUB_ENDPOINT", ""),
		HubKey:       getEnv("HW_HUB_KEY", ""),
		ConfigRev:    getEnv("HW_CONFIG_REV", ""),
		TestMode:     getEnv("HW_TEST_MODE", "false") == "true",
		client:       &http.Client{Timeout: 10 * time.Second},
		eventCh:      make(chan map[string]any, 1000),
		stopCh:       make(chan struct{}),
		// codeql[go/insecure-randomness] Non-cryptographic use case.
		// nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	if s.HubEndpoint == "" || s.HubKey == "" || s.SensorID == "" {
		return nil, fmt.Errorf("missing vars: HW_HUB_ENDPOINT, HW_HUB_KEY, HW_SENSOR_ID")
	}

	return s, nil
}

// SetTestPayload allows community sensors to define a realistic, protocol-specific mock payload
func (s *Sensor) SetTestPayload(trigger, source, target string, details map[string]any) {
	s.testPayload = map[string]any{
		"eventTrigger": trigger,
		"source":       source,
		"target":       target,
		"details":      details,
	}
	s.testTrigger = trigger
	s.testSource = source
	s.testTarget = target
	s.testDetails = details
}

func (s *Sensor) Start() error {
	if err := s.syncHubVersion(); err != nil {
		return err
	}
	go s.eventLoop()
	go s.heartbeatLoop()
	go s.listenForSignals()
	return nil
}

func (s *Sensor) Stop() {
	close(s.stopCh)
	s.GoOffline("graceful_shutdown")
}

func (s *Sensor) listenForSignals() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR1)
	for {
		select {
		case <-sigCh:
			log.Println("[*] SIGUSR1 received: injecting test event into queue...")
			trigger := "test_mode_synthetic_alert"
			source := "Wizard Live Test"
			target := "Mock Hub"
			details := map[string]any{"test_message": "Wizard triggered a live test event firedrill."}

			if s.testTrigger != "" {
				trigger = s.testTrigger
				source = s.testSource
				target = s.testTarget
				details = s.testDetails
			}
			s.ReportEvent(trigger, source, target, details)
		case <-s.stopCh:
			return
		}
	}
}

func (s *Sensor) RunTestMode() bool {
	log.Println("[*] Test mode: sending synthetic payload...")

	// 1. Establish handshake to fetch the contract version
	if err := s.syncHubVersion(); err != nil {
		log.Printf("[-] Test mode sync failed: %v", err)
		return false
	}

	trigger := "test_mode_synthetic_alert"
	source := "CI/CD Runner"
	target := "Mock Hub"
	details := map[string]any{"test_message": "Automated CI/CD check."}

	if s.testTrigger != "" {
		trigger = s.testTrigger
		source = s.testSource
		target = s.testTarget
		details = s.testDetails
	}

	// 2. Synchronously send the payload to guarantee delivery before the program exits
	payload := map[string]any{
		"contractVersion": s.hubContractVersion,
		"sensorId":        s.SensorID,
		"severity":        s.Severity,
		"eventTrigger":    trigger,
		"source":          source,
		"target":          target,
		"details":         details,
	}

	resp, err := s.postToHub("/api/v1/event", payload)
	if err != nil {
		log.Printf("[-] Test mode failed to send event: %v", err)
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode < 400
}

// ==========================================
// PIPELINE A: EVENT WORKER
// ==========================================

func (s *Sensor) ReportEvent(trigger, source, target string, details map[string]any) bool {
	payload := map[string]any{
		"contractVersion": s.hubContractVersion,
		"sensorId":        s.SensorID,
		"severity":        s.Severity,
		"eventTrigger":    trigger,
		"source":          source,
		"target":          target,
		"details":         details,
	}

	select {
	case s.eventCh <- payload:
		return true
	default:
		log.Printf("[-] Event buffer full. Dropping event.")
		return false
	}
}

func (s *Sensor) eventLoop() {
	for {
		select {
		case <-s.stopCh:
			s.drainQueue()
			return
		case event := <-s.eventCh:
			s.processEvent(event)
		}
	}
}

func (s *Sensor) processEvent(event map[string]any) {
	for attempt := 0; attempt < MaxRetriesPerEvent; attempt++ {
		resp, err := s.postToHub("/api/v1/event", event)
		fact := classify(err, resp)

		if resp != nil {
			resp.Body.Close()
		}

		action, delay := s.eventPolicy(fact, attempt)

		switch action {
		case EventSuccess:
			log.Printf("[+] Event reported successfully.")
			return
		case EventDrop:
			log.Printf("[-] Terminal failure (HTTP %d). Dropping poison event.", fact.StatusCode)
			return
		case EventRetry:
			log.Printf("[!] Transient issue. Retrying event (%d/%d) in %v...", attempt+1, MaxRetriesPerEvent, delay)
			t := time.NewTimer(delay)
			select {
			case <-t.C:
				continue
			case <-s.stopCh:
				t.Stop()
				return
			}
		}
	}
	log.Printf("[-] Event exceeded MaxRetriesPerEvent. Dropped.")
}

// Best effort flush of remaining events on shutdown signal
func (s *Sensor) drainQueue() {
	for {
		select {
		case event := <-s.eventCh:
			if resp, err := s.postToHub("/api/v1/event", event); err == nil && resp != nil {
				resp.Body.Close()
			}
		default:
			return
		}
	}
}

// ==========================================
// PIPELINE B: HEARTBEAT WORKER
// ==========================================

func (s *Sensor) heartbeatLoop() {
	sleepDuration := time.Duration(0)

	for {
		if sleepDuration > 0 {
			t := time.NewTimer(sleepDuration)
			select {
			case <-t.C:
			case <-s.stopCh:
				t.Stop()
				return
			}
		}

		resp, err := s.sendHeartbeat()
		fact := classify(err, resp)

		if resp != nil {
			resp.Body.Close()
		}

		// The policy tells us exactly how long until the next pulse
		sleepDuration = s.heartbeatPolicy(fact)

		if fact.IsError {
			log.Printf("[!] Heartbeat degraded. Next pulse in %v", sleepDuration)
		}
	}
}

func (s *Sensor) sendHeartbeat() (*http.Response, error) {
	payload := map[string]any{
		"sensorId":        s.SensorID,
		"agentVersion":    s.AgentVersion,
		"contractVersion": s.hubContractVersion,
		"configRev":       s.ConfigRev,
	}
	return s.postToHub("/api/v1/heartbeat", payload)
}

// ==========================================
// UTILITIES
// ==========================================

func (s *Sensor) syncHubVersion() error {
	if s.hubContractVersion != "" {
		return nil // Short-circuit if already synced
	}

	for i := 0; i < 3; i++ {
		req, err := http.NewRequest("GET", s.HubEndpoint+"/api/v1/version", nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+s.HubKey)

		resp, err := s.client.Do(req)
		fact := classify(err, resp)

		if !fact.IsError {
			var result map[string]string
			decodeErr := json.NewDecoder(resp.Body).Decode(&result)
			resp.Body.Close()

			if decodeErr == nil {
				s.hubContractVersion = result["version"]
				return nil
			}
		} else if resp != nil {
			resp.Body.Close()
		}

		if !fact.IsTransient {
			return fmt.Errorf("fatal synchronization failure (HTTP %d)", fact.StatusCode)
		}

		time.Sleep(s.calculateBackoff(i))
	}
	return fmt.Errorf("failed to sync with Hub after backoff limits")
}

func (s *Sensor) GoOffline(reason string) {
	fastClient := &http.Client{Timeout: 2 * time.Second}
	payload := map[string]any{"sensorId": s.SensorID, "reason": reason}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", s.HubEndpoint+"/api/v1/offline", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.HubKey)

	if resp, err := fastClient.Do(req); err == nil {
		resp.Body.Close()
	}
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
