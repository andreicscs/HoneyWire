package notify

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
	"fmt"
	"strings"
)

var client = &http.Client{Timeout: 5 * time.Second}

func Dispatch(webhookType, webhookURL, title, message, severity string) {
	if webhookURL == "" {
		return
	}

	switch strings.ToLower(webhookType) {
	case "discord", "slack":
		go sendDiscordSlack(webhookURL, title, message, severity)
	case "gotify":
		go sendGotify(webhookURL, title, message, severity)
	case "ntfy":
		fallthrough
	default:
		go sendNtfy(webhookURL, title, message, severity)
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