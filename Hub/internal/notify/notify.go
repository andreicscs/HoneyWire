package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

var client = &http.Client{Timeout: 5 * time.Second}

type WebhookPayload struct {
	Type     string
	URL      string
	Title    string
	Message  string
	Severity string
	QueuedAt time.Time
}

type NotifyConfig struct {
	IsArmed       bool
	WebhookType   string
	WebhookURL    string
	WebhookEvents string
}

var CurrentConfig NotifyConfig

func UpdateConfig(isArmed bool, webhookType, webhookURL, webhookEvents string) {
	CurrentConfig.IsArmed = isArmed
	CurrentConfig.WebhookType = webhookType
	CurrentConfig.WebhookURL = webhookURL
	CurrentConfig.WebhookEvents = webhookEvents
}

func UpdateIsArmed(isArmed bool) {
	CurrentConfig.IsArmed = isArmed
}

var WebhookQueue = make(chan WebhookPayload, 1000)

func StartWorker() {
	go func() {
		for payload := range WebhookQueue {
			sendWebhook(payload)
			time.Sleep(500 * time.Millisecond)
		}
	}()
	log.Println("[Notify] Worker started.")
}

func Dispatch(title, message, severity string) {
	if !CurrentConfig.IsArmed || CurrentConfig.WebhookURL == "" {
		return
	}

	if !strings.Contains(strings.ToLower(CurrentConfig.WebhookEvents), strings.ToLower(severity)) {
		return
	}

	payload := WebhookPayload{
		Type:     CurrentConfig.WebhookType,
		URL:      CurrentConfig.WebhookURL,
		Title:    title,
		Message:  message,
		Severity: severity,
		QueuedAt: time.Now(),
	}

	select {
	case WebhookQueue <- payload:
	default:
		log.Println("[!] Webhook queue full, dropping notification")
	}
}

func sendWebhook(payload WebhookPayload) {
	switch strings.ToLower(payload.Type) {
	case "discord", "slack":
		sendDiscordSlack(payload.URL, payload.Title, payload.Message, payload.Severity)
	case "gotify":
		sendGotify(payload.URL, payload.Title, payload.Message, payload.Severity)
	case "ntfy":
		fallthrough
	default:
		sendNtfy(payload.URL, payload.Title, payload.Message, payload.Severity)
	}
}

func sendGotify(webhookURL, title, message, severity string) {
	priorities := map[string]int{"info": 1, "low": 3, "medium": 5, "high": 8, "critical": 10}
	priority, exists := priorities[severity]
	if !exists {
		priority = 5
	}

	payload := map[string]interface{}{
		"title":    title,
		"message":  message,
		"priority": priority,
	}
	body, _ := json.Marshal(payload)

	// If the user pasted the Gotify URL as https://gotify.domain.com/message?token=XYZ,
	// standard POSTing to it natively works without needing the separate Header.
	req, _ := http.NewRequest("POST", webhookURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Gotify connection failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("Gotify rejected request. Status: %d", resp.StatusCode)
	}
}

func sendNtfy(webhookURL, title, message, severity string) {
	priorities := map[string]string{"info": "1", "low": "2", "medium": "3", "high": "4", "critical": "5"}
	priority, exists := priorities[severity]
	if !exists {
		priority = "3"
	}

	req, _ := http.NewRequest("POST", webhookURL, strings.NewReader(message))
	req.Header.Set("Title", title)
	req.Header.Set("Priority", priority)
	req.Header.Set("Tags", "rotating_light")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Ntfy connection failed: %v", err)
		return
	}
	defer resp.Body.Close()
}

func sendDiscordSlack(webhookURL, title, message, severity string) {
	// Discord and Slack both accept basic JSON payloads with a "content" string.
	icon := "⚠️"
	if severity == "critical" || severity == "high" {
		icon = "🚨"
	}

	payload := map[string]interface{}{
		"content": fmt.Sprintf("%s **%s**\n%s", icon, title, message),
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", webhookURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Discord/Slack connection failed: %v", err)
		return
	}
	defer resp.Body.Close()
}

func FlushQueue(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		select {
		case payload := <-WebhookQueue:
			sendWebhook(payload)
		case <-time.After(100 * time.Millisecond):
			select {
			case payload := <-WebhookQueue:
				sendWebhook(payload)
			default:
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("webhook flush timeout exceeded")
		}
	}
}
