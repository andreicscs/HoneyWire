package sdk

import (
	"errors"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

// TestClassify verifies the factual extraction of HTTP responses
func TestClassify(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		statusCode   int
		retryAfter   string
		wantError    bool
		wantTrans    bool
		wantRetryDur time.Duration
	}{
		{"Network Error", errors.New("timeout"), 0, "", true, true, 0},
		{"Success OK", nil, 200, "", false, false, 0},
		{"Success Created", nil, 201, "", false, false, 0},
		{"Bad Request (Terminal)", nil, 400, "", true, false, 0},
		{"Unauthorized (Terminal)", nil, 401, "", true, false, 0},
		{"Not Found (Terminal)", nil, 404, "", true, false, 0},
		{"Rate Limit (Transient)", nil, 429, "10", true, true, 10 * time.Second},
		{"Malformed Retry-After", nil, 429, "potato", true, true, 0},
		{"Server Error (Transient)", nil, 500, "", true, true, 0},
		{"Bad Gateway (Transient)", nil, 502, "30", true, true, 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *http.Response
			if tt.err == nil {
				resp = &http.Response{
					StatusCode: tt.statusCode,
					Header:     make(http.Header),
				}
				if tt.retryAfter != "" {
					resp.Header.Set("Retry-After", tt.retryAfter)
				}
			}

			fact := classify(tt.err, resp)

			if fact.IsError != tt.wantError {
				t.Errorf("IsError = %v, want %v", fact.IsError, tt.wantError)
			}
			if fact.IsTransient != tt.wantTrans {
				t.Errorf("IsTransient = %v, want %v", fact.IsTransient, tt.wantTrans)
			}
			if fact.RetryAfter != tt.wantRetryDur {
				t.Errorf("RetryAfter = %v, want %v", fact.RetryAfter, tt.wantRetryDur)
			}
		})
	}
}

// TestEventPolicy verifies that the event retry logic adheres to state transitions
func TestEventPolicy(t *testing.T) {
	s := &Sensor{}

	// Case 1: Success
	action, _ := s.eventPolicy(ResponseFact{IsError: false}, 0)
	if action != EventSuccess {
		t.Errorf("Expected EventSuccess, got %v", action)
	}

	// Case 2: Terminal Error
	action, _ = s.eventPolicy(ResponseFact{IsError: true, IsTransient: false}, 0)
	if action != EventDrop {
		t.Errorf("Expected EventDrop, got %v", action)
	}

	// Case 3: Transient Error with calculated backoff
	action, delay := s.eventPolicy(ResponseFact{IsError: true, IsTransient: true}, 0)
	if action != EventRetry {
		t.Errorf("Expected EventRetry, got %v", action)
	}
	if delay <= 0 {
		t.Errorf("Expected positive delay for retry, got %v", delay)
	}

	// Case 4: Transient Error with explicit Retry-After override
	action, delay = s.eventPolicy(ResponseFact{IsError: true, IsTransient: true, RetryAfter: 15 * time.Second}, 0)
	if action != EventRetry || delay != 15*time.Second {
		t.Errorf("Expected EventRetry with 15s delay, got %v with %v", action, delay)
	}
}

// TestHeartbeatPolicy verifies the heartbeat pulse intervals
func TestHeartbeatPolicy(t *testing.T) {
	s := &Sensor{}

	// Case 1: Normal Operation
	delay := s.heartbeatPolicy(ResponseFact{IsError: false})
	if delay != BaseHeartbeatInterval {
		t.Errorf("Expected BaseHeartbeatInterval (%v), got %v", BaseHeartbeatInterval, delay)
	}

	// Case 2: Terminal Error (Cooldown)
	delay = s.heartbeatPolicy(ResponseFact{IsError: true, IsTransient: false})
	if delay != TerminalSleepInterval {
		t.Errorf("Expected TerminalSleepInterval (%v), got %v", TerminalSleepInterval, delay)
	}

	// Case 3: Rate Limited (Valid Wait)
	delay = s.heartbeatPolicy(ResponseFact{IsError: true, IsTransient: true, RetryAfter: 45 * time.Second})
	if delay != 45*time.Second {
		t.Errorf("Expected 45s RetryAfter, got %v", delay)
	}

	// Case 4: Rate Limited (Short Retry-After ignored)
	delay = s.heartbeatPolicy(ResponseFact{IsError: true, IsTransient: true, RetryAfter: 10 * time.Second})
	if delay != BaseHeartbeatInterval {
		t.Errorf("Expected BaseHeartbeatInterval (%v) ignoring short Retry-After, got %v", BaseHeartbeatInterval, delay)
	}
}

// TestReportEvent_BufferLimit verifies the non-blocking channel dropping mechanism
func TestReportEvent_BufferLimit(t *testing.T) {
	s := &Sensor{
		SensorID: "test-sensor",
		eventCh:  make(chan map[string]any, 2), // Initialize with an artificially small buffer
	}

	// Fill the buffer
	ok1 := s.ReportEvent("trigger_1", "src", "tgt", nil)
	ok2 := s.ReportEvent("trigger_2", "src", "tgt", nil)

	// The third event should be dropped immediately
	ok3 := s.ReportEvent("trigger_3", "src", "tgt", nil)

	if !ok1 || !ok2 {
		t.Errorf("Expected first two events to be buffered successfully")
	}
	if ok3 {
		t.Errorf("Expected third event to be dropped due to full buffer")
	}
}

// TestCalculateBackoff verifies that the backoff delay increases with attempts
// and respects the maximum cap, despite random jitter.
func TestCalculateBackoff(t *testing.T) {
	s := &Sensor{}
	d0 := s.calculateBackoff(0)
	d1 := s.calculateBackoff(1)
	d2 := s.calculateBackoff(2)

	if d1 <= d0 {
		t.Errorf("Expected delay to increase: d1 (%v) <= d0 (%v)", d1, d0)
	}
	if d2 <= d1 {
		t.Errorf("Expected delay to increase: d2 (%v) <= d1 (%v)", d2, d1)
	}

	// Test cap (max delay is 60s base + up to 10% jitter = ~66s max)
	dMax := s.calculateBackoff(10)
	if dMax > 66*time.Second {
		t.Errorf("Expected max delay cap around 60-66s, got %v", dMax)
	}
}

// TestProcessEvent_RetryCap verifies that transient events are retried up to the limit then dropped.
func TestProcessEvent_RetryCap(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Retry-After", "-1") // bypass exponential backoff sleep
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := &Sensor{
		HubEndpoint: srv.URL,
		HubKey:      "test-key",
		client:      srv.Client(),
		stopCh:      make(chan struct{}),
	}

	s.processEvent(map[string]any{"test": "event"})

	if attempts != MaxRetriesPerEvent {
		t.Errorf("Expected exactly %d retries, got %d", MaxRetriesPerEvent, attempts)
	}
}

// TestSyncHubVersion_TerminalFailure verifies that terminal failures halt immediately.
func TestSyncHubVersion_TerminalFailure(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusUnauthorized) // 401 Terminal
	}))
	defer srv.Close()

	s := &Sensor{
		HubEndpoint: srv.URL,
		HubKey:      "bad-key",
		client:      srv.Client(),
	}

	err := s.syncHubVersion()
	if err == nil {
		t.Errorf("Expected terminal failure error, got nil")
	}

	if attempts != 1 {
		t.Errorf("Expected exactly 1 attempt before terminal failure, got %d", attempts)
	}
}

// TestReportEvent_Serialization verifies that a reported event is correctly
// serialized and sent to the hub via the event loop.
func TestReportEvent_Serialization(t *testing.T) {
	// Use a channel to signal from the server handler back to the test
	payloadReceived := make(chan map[string]any, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/event" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Server failed to decode payload: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		payloadReceived <- payload
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Manually construct the sensor to ensure a hermetic test
	s := &Sensor{
		SensorID:           "test-sensor-123",
		Severity:           "high",
		hubContractVersion: "1.0.0-test",
		HubEndpoint:        srv.URL,
		HubKey:             "test-key",
		client:             srv.Client(),
		eventCh:            make(chan map[string]any, 10),
		stopCh:             make(chan struct{}),
	}

	// Start the worker that pulls from the channel and posts to the hub
	go s.eventLoop()
	defer s.Stop() // Ensure the goroutine is cleaned up

	// The event data we expect to see on the server
	trigger := "test_serialization"
	source := "192.168.1.10"
	target := "8.8.8.8:53"
	details := map[string]any{"protocol": "dns"}

	// Report the event, which places it on the channel
	if !s.ReportEvent(trigger, source, target, details) {
		t.Fatal("ReportEvent failed to buffer the event")
	}

	// Wait for the server to receive the payload, with a timeout
	select {
	case receivedPayload := <-payloadReceived:
		// Assertions on the received payload
		if got, want := receivedPayload["sensorId"], s.SensorID; got != want {
			t.Errorf("sensorId mismatch: got %v, want %v", got, want)
		}
		if got, want := receivedPayload["eventTrigger"], trigger; got != want {
			t.Errorf("eventTrigger mismatch: got %v, want %v", got, want)
		}
		if got, want := receivedPayload["source"], source; got != want {
			t.Errorf("source mismatch: got %v, want %v", got, want)
		}
		if got, want := receivedPayload["target"], target; got != want {
			t.Errorf("target mismatch: got %v, want %v", got, want)
		}
		if !reflect.DeepEqual(receivedPayload["details"], details) {
			t.Errorf("details mismatch: got %v, want %v", receivedPayload["details"], details)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for server to receive event payload")
	}
}

// TestDrainQueueOnStop verifies that pending events are flushed on shutdown.
func TestDrainQueueOnStop(t *testing.T) {
	// Use a channel to count event requests received by the server.
	eventRequests := make(chan struct{}, 5)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/event":
			eventRequests <- struct{}{}
			w.WriteHeader(http.StatusOK)
		case "/api/v1/offline":
			// Acknowledge the offline message sent by s.Stop()
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	s := &Sensor{
		SensorID:    "test-drain-sensor",
		HubEndpoint: srv.URL,
		HubKey:      "test-key",
		client:      srv.Client(),
		eventCh:     make(chan map[string]any, 10),
		stopCh:      make(chan struct{}),
	}

	// Pre-load the event channel with some events
	const eventCount = 3
	for i := 0; i < eventCount; i++ {
		s.eventCh <- map[string]any{"id": i}
	}

	// Start the event loop in the background
	go s.eventLoop()

	// Trigger the shutdown, which should initiate the drain.
	s.Stop()

	// Verify that all pre-loaded events were drained and sent.
	for i := 0; i < eventCount; i++ {
		select {
		case <-eventRequests:
			// Good, one request was received.
		case <-time.After(1 * time.Second):
			t.Fatalf("Timed out waiting for drained event %d of %d", i+1, eventCount)
		}
	}

	// Check if there are any extra requests, which would be an error.
	if len(eventRequests) > 0 {
		t.Errorf("Received %d more events than expected", len(eventRequests))
	}
}
