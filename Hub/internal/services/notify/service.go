package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"

	// codeql[go/insecure-randomness] Non-cryptographic use case.
	// nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
	"math/rand"
	"github.com/honeywire/hub/internal/models"
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

func (s *Service) calculateBackoff(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return retryAfter
	}
	base := 2.0
	maxDelay := 120.0
	delay := base * math.Pow(2, float64(attempt))
	if delay > maxDelay {
		delay = maxDelay
	}
	jitter := (s.rng.Float64() * 0.2) - 0.1
	return time.Duration((delay + (delay * jitter)) * float64(time.Second))
}

// ============================================================================
// SERVICE CORE
// ============================================================================

type WebhookPayload struct {
	Type     string
	URL      string
	Title    string
	Trigger  string
	SensorID string
	Source   string
	Target   string
	Time     string
	Severity string
	QueuedAt time.Time
}

type NodeService interface {
	GetNodeDetails(nodeID string) (*models.Node, error)
}

type Service struct {
	isArmed        bool
	webhookType    string
	webhookURL     string
	severityFilter map[string]struct{}

	mu           sync.RWMutex
	webhookQueue chan WebhookPayload

	client      *http.Client
	wg          sync.WaitGroup
	isDraining  atomic.Bool
	rng         *rand.Rand
	nodeService NodeService
}

func NewService(nodeService NodeService) *Service {
	return &Service{
		webhookQueue:   make(chan WebhookPayload, 1000),
		severityFilter: make(map[string]struct{}),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		// codeql[go/insecure-randomness] Non-cryptographic use case.
		// nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
		nodeService: nodeService,
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

func (s *Service) Dispatch(e models.Event) {
	if s.isDraining.Load() {
		return
	}

	s.mu.RLock()
	isArmed := s.isArmed
	webhookURL := s.webhookURL
	webhookType := s.webhookType
	_, allowed := s.severityFilter[strings.ToLower(e.Severity)]
	s.mu.RUnlock()

	if !isArmed || webhookURL == "" || !allowed {
		return
	}

	nodeAlias := e.NodeID
	if s.nodeService != nil {
		if nodeDetails, err := s.nodeService.GetNodeDetails(e.NodeID); err == nil && nodeDetails != nil {
			if nodeDetails.Alias != "" {
				nodeAlias = nodeDetails.Alias
			}
		}
	}

	// Unified title: "Threat detected on <node>"
	title := fmt.Sprintf("Threat detected on %s", nodeAlias)

	payload := WebhookPayload{
		Type:     webhookType,
		URL:      webhookURL,
		Title:    title,
		Trigger:  e.EventTrigger,
		SensorID: e.SensorID,
		Source:   e.Source,
		Target:   e.Target,
		Time:     time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
		Severity: e.Severity,
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

		delay := s.calculateBackoff(attempt, fact.RetryAfter)
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
			log.Println("[Notify] Worker stopped.")
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
			log.Println("[Notify] Worker stopped.")
			return
		}
	}
}

// ============================================================================
// SENDERS (Returning HTTP Responses)
// ============================================================================

func (s *Service) executeSend(payload WebhookPayload) (*http.Response, error) {
	switch strings.ToLower(payload.Type) {
	case "discord":
		return s.sendDiscord(payload)
	case "slack":
		return s.sendSlack(payload)
	case "gotify":
		return s.sendGotify(payload)
	case "ntfy":
		fallthrough
	default:
		return s.sendNtfy(payload)
	}
}

func (s *Service) sendGotify(payload WebhookPayload) (*http.Response, error) {
	priorities := map[string]int{
		"info": 1, "low": 3, "medium": 5, "high": 8, "critical": 10,
	}
	priority, ok := priorities[strings.ToLower(payload.Severity)]
	if !ok {
		priority = 5
	}

	msg := plainTextMessage(payload)
	reqPayload := map[string]interface{}{
		"title":    payload.Title,
		"message":  msg,
		"priority": priority,
	}
	body, _ := json.Marshal(reqPayload)

	req, _ := http.NewRequest("POST", payload.URL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	return s.client.Do(req)
}

func (s *Service) sendNtfy(payload WebhookPayload) (*http.Response, error) {
	priorities := map[string]string{
		"info": "1", "low": "2", "medium": "3", "high": "4", "critical": "5",
	}
	priority, ok := priorities[strings.ToLower(payload.Severity)]
	if !ok {
		priority = "3"
	}

	msg := plainTextMessage(payload)
	req, _ := http.NewRequest("POST", payload.URL, strings.NewReader(msg))
	req.Header.Set("Title", payload.Title)
	req.Header.Set("Priority", priority)
	req.Header.Set("Tags", ntfyTags(payload.Severity))
	return s.client.Do(req)
}

func (s *Service) sendDiscord(payload WebhookPayload) (*http.Response, error) {
	msg := discordMessage(payload)
	reqPayload := map[string]interface{}{
		"username": "HoneyWire",
		"embeds": []map[string]interface{}{
			{
				"title":       payload.Title,
				"description": msg,
				"color":       severityColor(payload.Severity),
				"footer": map[string]string{
					"text": fmt.Sprintf("Severity: %s", strings.ToUpper(payload.Severity)),
				},
			},
		},
	}
	body, _ := json.Marshal(reqPayload)

	req, _ := http.NewRequest("POST", payload.URL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	return s.client.Do(req)
}

func (s *Service) sendSlack(payload WebhookPayload) (*http.Response, error) {
	msg := slackMessage(payload)
	reqPayload := map[string]interface{}{
		"text": payload.Title,
		"attachments": []map[string]interface{}{
			{
				"color":  severityColorHex(payload.Severity),
				"title":  payload.Title,
				"text":   msg,
				"footer": fmt.Sprintf("Severity: %s", strings.ToUpper(payload.Severity)),
			},
		},
	}
	body, _ := json.Marshal(reqPayload)

	req, _ := http.NewRequest("POST", payload.URL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	return s.client.Do(req)
}

// ============================================================================
// FORMATTING HELPERS
// ============================================================================

func plainTextMessage(p WebhookPayload) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Trigger: %s\n", p.Trigger))
	b.WriteString(fmt.Sprintf("Sensor: %s\n", p.SensorID))
	if p.Source != "" {
		b.WriteString(fmt.Sprintf("Source: %s\n", p.Source))
	}
	if p.Target != "" {
		b.WriteString(fmt.Sprintf("Target: %s\n", p.Target))
	}
	b.WriteString(fmt.Sprintf("Time: %s", p.Time))
	return b.String()
}

func discordMessage(p WebhookPayload) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("**Trigger:** %s\n", p.Trigger))
	b.WriteString(fmt.Sprintf("**Sensor:** %s\n", p.SensorID))
	if p.Source != "" {
		b.WriteString(fmt.Sprintf("**Source:** %s\n", p.Source))
	}
	if p.Target != "" {
		b.WriteString(fmt.Sprintf("**Target:** %s\n", p.Target))
	}
	b.WriteString(fmt.Sprintf("**Time:** %s", p.Time))
	return b.String()
}

func slackMessage(p WebhookPayload) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("*Trigger:* %s\n", p.Trigger))
	b.WriteString(fmt.Sprintf("*Sensor:* %s\n", p.SensorID))
	if p.Source != "" {
		b.WriteString(fmt.Sprintf("*Source:* %s\n", p.Source))
	}
	if p.Target != "" {
		b.WriteString(fmt.Sprintf("*Target:* %s\n", p.Target))
	}
	b.WriteString(fmt.Sprintf("*Time:* %s", p.Time))
	return b.String()
}

// severityColor returns a decimal embed color for Discord per severity.
func severityColor(severity string) int {
	switch strings.ToLower(severity) {
	case "critical":
		return 0xE24B4A // red
	case "high":
		return 0xD85A30 // coral
	case "medium":
		return 0xBA7517 // amber
	case "low":
		return 0x378ADD // blue
	default:
		return 0x888780 // gray
	}
}

// severityColorHex returns a hex color string for Slack attachments per severity.
func severityColorHex(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "#E24B4A"
	case "high":
		return "#D85A30"
	case "medium":
		return "#BA7517"
	case "low":
		return "#378ADD"
	default:
		return "#888780"
	}
}

// ntfyTags returns a comma-separated tag string for ntfy per severity.
func ntfyTags(severity string) string {
	base := "rotating_light,honeywire"
	switch strings.ToLower(severity) {
	case "critical", "high":
		return base + ",warning"
	default:
		return base
	}
}

func (s *Service) Wait() {
	s.wg.Wait()
}
