package notify

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/honeywire/hub/internal/models"
)

type mockNodeService struct{}

func (m *mockNodeService) GetNodeDetails(nodeID string) (*models.Node, error) {
	return &models.Node{
		ID:    nodeID,
		Alias: "Production-Server",
	}, nil
}

func TestNotify_NtfyFormatting(t *testing.T) {
	var receivedBody string
	var receivedTitle string
	var receivedPriority string
	var receivedTags string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		receivedBody = string(b)
		receivedTitle = r.Header.Get("Title")
		receivedPriority = r.Header.Get("Priority")
		receivedTags = r.Header.Get("Tags")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	svc := NewService(&mockNodeService{})
	svc.UpdateConfig(true, "ntfy", ts.URL, "critical,high")

	evt := models.Event{
		NodeID:       "node1",
		SensorID:     "hw-sensor-file-canary",
		EventTrigger: "file_tamper",
		Severity:     "critical",
		Source:       "192.168.1.100",
		Target:       "/etc/passwd",
	}

	payload := WebhookPayload{
		Type:     "ntfy",
		URL:      ts.URL,
		Title:    "Threat detected on Production-Server",
		Trigger:  evt.EventTrigger,
		SensorID: evt.SensorID,
		Source:   evt.Source,
		Target:   evt.Target,
		Time:     "2026-08-28 15:00:00 UTC",
		Severity: evt.Severity,
		QueuedAt: time.Now(),
	}

	resp, err := svc.executeSend(payload)
	if err != nil {
		t.Fatalf("executeSend failed: %v", err)
	}
	resp.Body.Close()

	// Verify ntfy received plain text without markdown asterisks
	if strings.Contains(receivedBody, "**") {
		t.Errorf("ntfy body should not contain '**' markdown asterisks, got: %s", receivedBody)
	}
	if !strings.Contains(receivedBody, "Trigger: file_tamper") {
		t.Errorf("ntfy body missing 'Trigger: file_tamper', got: %s", receivedBody)
	}
	if !strings.Contains(receivedBody, "Source: 192.168.1.100") {
		t.Errorf("ntfy body missing 'Source: 192.168.1.100', got: %s", receivedBody)
	}
	if receivedTitle != "Threat detected on Production-Server" {
		t.Errorf("Expected title 'Threat detected on Production-Server', got: %s", receivedTitle)
	}
	if receivedPriority != "5" {
		t.Errorf("Expected critical priority '5', got: %s", receivedPriority)
	}
	if !strings.Contains(receivedTags, "warning") {
		t.Errorf("Expected tags to contain 'warning', got: %s", receivedTags)
	}
}

func TestNotify_DiscordFormatting(t *testing.T) {
	var receivedJSON map[string]interface{}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedJSON)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	svc := NewService(&mockNodeService{})
	payload := WebhookPayload{
		Type:     "discord",
		URL:      ts.URL,
		Title:    "Threat detected on Production-Server",
		Trigger:  "port_scan",
		SensorID: "hw-sensor-network-scan-detector",
		Severity: "high",
		Time:     "2026-08-28 15:00:00 UTC",
	}

	resp, err := svc.executeSend(payload)
	if err != nil {
		t.Fatalf("executeSend failed: %v", err)
	}
	resp.Body.Close()

	embeds, ok := receivedJSON["embeds"].([]interface{})
	if !ok || len(embeds) == 0 {
		t.Fatalf("Expected embeds in Discord payload, got: %v", receivedJSON)
	}
	embed := embeds[0].(map[string]interface{})
	desc := embed["description"].(string)

	if !strings.Contains(desc, "**Trigger:** port_scan") {
		t.Errorf("Discord embed description should contain '**Trigger:** port_scan', got: %s", desc)
	}
}

func TestNotify_SlackFormatting(t *testing.T) {
	var receivedJSON map[string]interface{}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedJSON)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	svc := NewService(&mockNodeService{})
	payload := WebhookPayload{
		Type:     "slack",
		URL:      ts.URL,
		Title:    "Threat detected on Production-Server",
		Trigger:  "credential_trap",
		SensorID: "hw-sensor-tcp-tarpit",
		Severity: "critical",
		Time:     "2026-08-28 15:00:00 UTC",
	}

	resp, err := svc.executeSend(payload)
	if err != nil {
		t.Fatalf("executeSend failed: %v", err)
	}
	resp.Body.Close()

	attachments, ok := receivedJSON["attachments"].([]interface{})
	if !ok || len(attachments) == 0 {
		t.Fatalf("Expected attachments in Slack payload, got: %v", receivedJSON)
	}
	att := attachments[0].(map[string]interface{})
	text := att["text"].(string)
	color := att["color"].(string)

	if !strings.Contains(text, "*Trigger:* credential_trap") {
		t.Errorf("Slack attachment text should contain '*Trigger:* credential_trap', got: %s", text)
	}
	if color != "#E24B4A" {
		t.Errorf("Expected critical color #E24B4A, got: %s", color)
	}
}
