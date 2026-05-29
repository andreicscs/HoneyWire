package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MaxRetriesPerAlert = 10
)

// ============================================================================
// CLASSIFIER & POLICY ENGINE
// ============================================================================

type ResponseFact struct {
	IsError     bool
	IsTransient bool
	StatusCode  int
	RetryAfter  time.Duration
}

func classify(err error, resp *http.Response) ResponseFact {
	if err != nil {
		return ResponseFact{IsError: true, IsTransient: true} // Network drop
	}

	fact := ResponseFact{
		StatusCode: resp.StatusCode,
		IsError:    resp.StatusCode >= 400,
	}

	if fact.IsError {
		switch resp.StatusCode {
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
			fact.IsTransient = false // Terminal errors. Bad config, dead URL.
		default:
			fact.IsTransient = true // 429 Rate Limits, 5xx Server Errors
			
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if sec, err := strconv.Atoi(ra); err == nil {
					fact.RetryAfter = time.Duration(sec) * time.Second
				}
			}
		}
	}
	return fact
}

func calculateBackoff(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return retryAfter
	}
	base := 2.0
	maxDelay := 120.0
	delay := base * math.Pow(2, float64(attempt))
	if delay > maxDelay {
		delay = maxDelay
	}
	jitter := (rand.Float64() * 0.2) - 0.1
	return time.Duration((delay + (delay * jitter)) * float64(time.Second))
}

// ============================================================================
// SERVICE CORE
// ============================================================================

type WebhookPayload struct {
	Type     string
	URL      string
	Title    string
	Message  string
	Severity string
	QueuedAt time.Time
}

type Service struct {
	isArmed        bool
	webhookType    string
	webhookURL     string
	severityFilter map[string]struct{}
	
	mu           sync.RWMutex
	webhookQueue chan WebhookPayload
	
	client       *http.Client
	wg           sync.WaitGroup
	isDraining   atomic.Bool
}

func NewService() *Service {
	rand.Seed(time.Now().UnixNano())
	return &Service{
		webhookQueue: make(chan WebhookPayload, 1000),
		severityFilter: make(map[string]struct{}),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *Service) UpdateConfig(isArmed bool, webhookType, webhookURL, webhookEvents string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.isArmed = isArmed
	s.webhookType = webhookType
	s.webhookURL = webhookURL
	
	// Parse O(1) map once at config load
	filter := make(map[string]struct{})
	for _, sev := range strings.Split(webhookEvents, ",") {
		cleanSev := strings.TrimSpace(strings.ToLower(sev))
		if cleanSev != "" {
			filter[cleanSev] = struct{}{}
		}
	}
	s.severityFilter = filter
}

func (s *Service) UpdateIsArmed(isArmed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isArmed = isArmed
}

func (s *Service) Dispatch(title, message, severity string) {
	if s.isDraining.Load() {
		return
	}

	s.mu.RLock()
	isArmed := s.isArmed
	webhookURL := s.webhookURL
	webhookType := s.webhookType
	_, allowed := s.severityFilter[strings.ToLower(severity)]
	s.mu.RUnlock()

	if !isArmed || webhookURL == "" || !allowed {
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

// ============================================================================
// WORKER & RETRY ENGINE
// ============================================================================

func (s *Service) StartWorker(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		log.Println("[Notify] Worker started.")

		rateLimiter := time.NewTicker(500 * time.Millisecond)
		defer rateLimiter.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("[Notify] Shutdown signal received. Draining queue...")
				s.drainQueue()
				return
			case payload := <-s.webhookQueue:
				<-rateLimiter.C 
				s.processWithRetry(ctx, payload)
			}
		}
	}()
}

func (s *Service) processWithRetry(ctx context.Context, payload WebhookPayload) {
	for attempt := 0; attempt < MaxRetriesPerAlert; attempt++ {
		resp, err := s.executeSend(payload)
		fact := classify(err, resp)

		if resp != nil {
			resp.Body.Close()
		}

		if !fact.IsError {
			return // Success
		}

		if !fact.IsTransient {
			log.Printf("[-] Terminal webhook error (HTTP %d). Dropping payload to avoid spamming.", fact.StatusCode)
			return
		}

		delay := calculateBackoff(attempt, fact.RetryAfter)
		log.Printf("[!] Webhook failed (%s). Retrying (%d/%d) in %v...", payload.Type, attempt+1, MaxRetriesPerAlert, delay)

		t := time.NewTimer(delay)
		select {
		case <-t.C:
			continue
		case <-ctx.Done():
			t.Stop()
			return // Context canceled mid-retry, abort and let drainQueue take over
		}
	}
	log.Printf("[-] Webhook exceeded MaxRetries (%d). Dropped.", MaxRetriesPerAlert)
}

func (s *Service) drainQueue() {
	s.isDraining.Store(true)

	timeout := time.After(5 * time.Second)
	count := 0

	for {
		select {
		case <-timeout:
			log.Printf("[Notify] Drain timeout reached. Dropped %d remaining alerts.", len(s.webhookQueue))
			return
		case payload := <-s.webhookQueue:
			// Best-effort send: no retries during shutdown, but we respect HTTP lifecycles
			resp, err := s.executeSend(payload)
			if resp != nil {
				resp.Body.Close()
			}
			if err == nil && resp != nil && resp.StatusCode < 400 {
				count++
			}
			
			// Maintain rate limit during drain
			time.Sleep(500 * time.Millisecond) 
		default:
			log.Printf("[Notify] Queue completely drained. Flushed %d alerts.", count)
			return
		}
	}
}

// ============================================================================
// SENDERS (Returning HTTP Responses)
// ============================================================================

func (s *Service) executeSend(payload WebhookPayload) (*http.Response, error) {
	switch strings.ToLower(payload.Type) {
	case "discord", "slack":
		return s.sendDiscordSlack(payload.URL, payload.Title, payload.Message, payload.Severity)
	case "gotify":
		return s.sendGotify(payload.URL, payload.Title, payload.Message, payload.Severity)
	case "ntfy":
		fallthrough
	default:
		return s.sendNtfy(payload.URL, payload.Title, payload.Message, payload.Severity)
	}
}

func (s *Service) sendGotify(webhookURL, title, message, severity string) (*http.Response, error) {
	priorities := map[string]int{"info": 1, "low": 3, "medium": 5, "high": 8, "critical": 10}
	priority, exists := priorities[severity]
	if !exists { priority = 5 }

	payload := map[string]interface{}{
		"title":    title,
		"message":  message,
		"priority": priority,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", webhookURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	return s.client.Do(req)
}

func (s *Service) sendNtfy(webhookURL, title, message, severity string) (*http.Response, error) {
	priorities := map[string]string{"info": "1", "low": "2", "medium": "3", "high": "4", "critical": "5"}
	priority, exists := priorities[severity]
	if !exists { priority = "3" }

	req, _ := http.NewRequest("POST", webhookURL, strings.NewReader(message))
	req.Header.Set("Title", title)
	req.Header.Set("Priority", priority)
	req.Header.Set("Tags", "rotating_light")

	return s.client.Do(req)
}

func (s *Service) sendDiscordSlack(webhookURL, title, message, severity string) (*http.Response, error) {
	icon := "⚠️"
	if severity == "critical" || severity == "high" { icon = "🚨" }

	payload := map[string]interface{}{
		"content": fmt.Sprintf("%s **%s**\n%s", icon, title, message),
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", webhookURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	return s.client.Do(req)
}

func (s *Service) Wait() {
	s.wg.Wait()
}