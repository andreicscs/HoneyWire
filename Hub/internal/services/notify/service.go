package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
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

type Service struct {
	isArmed       bool
	webhookType   string
	webhookURL    string
	webhookEvents string
	mu            sync.RWMutex
	webhookQueue  chan WebhookPayload
}

func NewService() *Service {
	return &Service{
		webhookQueue: make(chan WebhookPayload, 1000),
	}
}

func (s *Service) UpdateConfig(isArmed bool, webhookType, webhookURL, webhookEvents string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isArmed = isArmed
	s.webhookType = webhookType
	s.webhookURL = webhookURL
	s.webhookEvents = webhookEvents
}

func (s *Service) UpdateIsArmed(isArmed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isArmed = isArmed
}

func (s *Service) StartWorker(ctx context.Context) {
	log.Println("[Notify] Worker started.")
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("[Notify] Worker stopped.")
				return
			case payload := <-s.webhookQueue:
				s.sendWebhook(payload)
				time.Sleep(500 * time.Millisecond)
			}
		}
	}()
}

func (s *Service) Dispatch(title, message, severity string) {
	s.mu.RLock()
	isArmed := s.isArmed
	webhookURL := s.webhookURL
	webhookEvents := s.webhookEvents
	webhookType := s.webhookType
	s.mu.RUnlock()

	if !isArmed || webhookURL == "" {
		return
	}

	if !strings.Contains(strings.ToLower(webhookEvents), strings.ToLower(severity)) {
		return
	}

	payload := WebhookPayload{
		Type:     webhookType,
		URL:      webhookURL,
		Title:    title,
		Message:  message,
		Severity: severity,
		QueuedAt: time.Now(),
	}

	select {
	case s.webhookQueue <- payload:
	default:
		log.Println("[!] Webhook queue full, dropping notification")
	}
}

func (s *Service) sendWebhook(payload WebhookPayload) {
	switch strings.ToLower(payload.Type) {
	case "discord", "slack":
		s.sendDiscordSlack(payload.URL, payload.Title, payload.Message, payload.Severity)
	case "gotify":
		s.sendGotify(payload.URL, payload.Title, payload.Message, payload.Severity)
	case "ntfy":
		fallthrough
	default:
		s.sendNtfy(payload.URL, payload.Title, payload.Message, payload.Severity)
	}
}

func (s *Service) sendGotify(webhookURL, title, message, severity string) {
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

func (s *Service) sendNtfy(webhookURL, title, message, severity string) {
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

func (s *Service) sendDiscordSlack(webhookURL, title, message, severity string) {
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

func (s *Service) FlushQueue(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for len(s.webhookQueue) > 0 {
		if time.Now().After(deadline) {
			return fmt.Errorf("webhook flush timeout exceeded, %d events remaining", len(s.webhookQueue))
		}
		payload := <-s.webhookQueue
		s.sendWebhook(payload)
	}
	return nil
}
