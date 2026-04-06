package notify

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/honeywire/hub/internal/config"
)

var client = &http.Client{Timeout: 5 * time.Second}

func Dispatch(cfg *config.Config, title, message, severity string) {
	if cfg.NtfyURL != "" {
		go sendNtfy(cfg, title, message, severity)
	}
	if cfg.GotifyURL != "" && cfg.GotifyToken != "" {
		go sendGotify(cfg, title, message, severity)
	}
}

func sendGotify(cfg *config.Config, title, message, severity string) {
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

	req, _ := http.NewRequest("POST", cfg.GotifyURL, bytes.NewBuffer(body))
	req.Header.Set("X-Gotify-App-Token", cfg.GotifyToken)
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

func sendNtfy(cfg *config.Config, title, message, severity string) {
	priorities := map[string]string{"info": "1", "low": "2", "medium": "3", "high": "4", "critical": "5"}
	priority, exists := priorities[severity]
	if !exists {
		priority = "3"
	}

	req, _ := http.NewRequest("POST", cfg.NtfyURL, bytes.NewBufferString(message))
	req.Header.Set("Title", title)
	req.Header.Set("Priority", priority)
	req.Header.Set("Tags", "rotating_light")

	if cfg.NtfyURL != "" { // Only add auth if token exists (logic can be expanded here)
		// req.Header.Set("Authorization", "Bearer "+cfg.NtfyToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Ntfy connection failed: %v", err)
		return
	}
	defer resp.Body.Close()
}