package siem

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"

	// codeql[go/insecure-randomness] Non-cryptographic use case.
	// nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
	"math/rand"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/honeywire/hub/internal/models"
)

const (
	MaxRetriesPerEvent = 10
)

// ============================================================================
// 1. SHARED CLASSIFIER (Pure Network Truth Layer)
// ============================================================================

type NetworkFact struct {
	IsError     bool
	IsTransient bool
	Err         error
}

// classify inspects raw socket/dialer errors to determine system state.
func classify(err error) NetworkFact {
	if err == nil {
		return NetworkFact{IsError: false, IsTransient: false, Err: nil}
	}

	fact := NetworkFact{
		IsError:     true,
		IsTransient: true, // Default raw network errors (timeouts, refusions) to transient
		Err:         err,
	}

	// Check for known terminal configuration anomalies
	errStr := err.Error()
	if strings.Contains(errStr, "unknown network") ||
		strings.Contains(errStr, "invalid argument") {
		fact.IsTransient = false
	}

	return fact
}

// ============================================================================
// 2. POLICY INTERPRETER (Domain-Specific Security Logging Rules)
// ============================================================================

type SiemAction string

const (
	SiemSuccess SiemAction = "success"
	SiemRetry   SiemAction = "retry"
	SiemDrop    SiemAction = "drop"
)

func (s *Service) siemPolicy(fact NetworkFact, attempt int) (SiemAction, time.Duration) {
	if !fact.IsError {
		return SiemSuccess, 0
	}
	if !fact.IsTransient {
		return SiemDrop, 0
	}

	// Calculate robust exponential backoff + jitter for transient network drops
	base := 2.0
	maxDelay := 60.0
	delay := base * math.Pow(2, float64(attempt))
	if delay > maxDelay {
		delay = maxDelay
	}
	jitter := (s.rng.Float64() * 0.2) - 0.1
	finalDelay := time.Duration((delay + (delay * jitter)) * float64(time.Second))

	return SiemRetry, finalDelay
}

// ============================================================================
// SERVICE CORE & CONFIGURATION
// ============================================================================

type NodeService interface {
	GetNodeDetails(nodeID string) (*models.Node, error)
}

type Service struct {
	eventQueue  chan models.Event
	address     string
	protocol    string
	rng         *rand.Rand
	mu          sync.RWMutex
	wg          sync.WaitGroup
	isDraining  atomic.Bool
	nodeService NodeService
}

func NewService(nodeService NodeService) *Service {
	return &Service{
		eventQueue: make(chan models.Event, 5000),
		// codeql[go/insecure-randomness] Non-cryptographic use case.
		// nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
		nodeService: nodeService,
	}
}

func (s *Service) UpdateConfig(address, protocol string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if protocol == "" {
		protocol = "tcp"
	}
	s.address = address
	s.protocol = protocol
}

func (s *Service) QueueEvent(event models.Event) {
	if s.isDraining.Load() {
		return
	}

	select {
	case s.eventQueue <- event:
	default:
		log.Println("[!] SIEM Pipeline saturated. Bounded queue full, dropping event.")
	}
}

// ============================================================================
// WORKER ENGINE (Persistent Streams + Non-Stalling Cooldowns)
// ============================================================================

// streamSession encapsulates the active connection state for a single worker goroutine.
// By binding methods to this struct, we eliminate pointer indirection and parameter bloat.
// not safe for concurrent access.
type streamSession struct {
	conn                 net.Conn
	currentAddr          string
	currentProto         string
	nextReconnectAttempt time.Time
	service              *Service
}

// close cleanly tears down the connection
func (sess *streamSession) close() {
	if sess.conn != nil {
		sess.conn.Close()
		sess.conn = nil
	}
}

func (s *Service) StartWorker(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		log.Println("[SIEM] Worker started. Listening for telemetry streams...")

		// Initialize the stateful session object
		sess := &streamSession{
			service: s,
		}
		defer sess.close()

		for {
			select {
			case <-ctx.Done():
				log.Println("[SIEM] Shutdown signal received. Flushing pipeline...")
				s.drainQueue(sess.conn)
				return

			case event := <-s.eventQueue:
				sess.processEventWithRetry(ctx, event)
			}
		}
	}()
}

func (sess *streamSession) processEventWithRetry(ctx context.Context, event models.Event) {
	msg := sess.service.formatSyslog(event)

	for attempt := 0; attempt < MaxRetriesPerEvent; attempt++ {
		sess.service.mu.RLock()
		addr, proto := sess.service.address, sess.service.protocol
		sess.service.mu.RUnlock()

		if addr == "" {
			return // Exporter explicitly disabled via configuration
		}

		// Handle Stream Synchronization/Instantiation cleanly
		if sess.conn == nil || addr != sess.currentAddr || proto != sess.currentProto {
			// Prevent aggressive dial storms against a dead receiver
			if time.Now().Before(sess.nextReconnectAttempt) {
				// Cooldown active: wait out the balance using a precise context-aware timer
				t := time.NewTimer(time.Until(sess.nextReconnectAttempt))
				select {
				case <-t.C:
				case <-ctx.Done():
					t.Stop()
					return
				}
			}

			sess.close()
			dialConn, err := net.DialTimeout(proto, addr, 5*time.Second)
			fact := classify(err)

			if fact.IsError {
				log.Printf("[!] SIEM Stream connection failed (%s://%s): %v", proto, addr, err)
				sess.nextReconnectAttempt = time.Now().Add(2 * time.Second) // 2-second suppression cooldown

				action, delay := sess.service.siemPolicy(fact, attempt)
				if action == SiemDrop {
					log.Printf("[-] Terminal configuration error. Dropping log stream position.")
					return
				}

				// Backoff and retry dialing for the exact same event log position
				if sess.service.sleepContext(ctx, delay) {
					return
				}
				continue
			}

			sess.conn = dialConn
			sess.currentAddr, sess.currentProto = addr, proto
		}

		// Stream Writing Phase
		sess.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		_, writeErr := sess.conn.Write([]byte(msg + "\n"))
		fact := classify(writeErr)
		action, delay := sess.service.siemPolicy(fact, attempt)

		switch action {
		case SiemSuccess:
			return // Log exported successfully. Advance queue position.
		case SiemDrop:
			log.Printf("[-] Terminal failure encountered on write. Discarding position.")
			sess.close()
			return
		case SiemRetry:
			log.Printf("[!] SIEM stream disconnected mid-write: %v. Retrying connection structure in %v...", writeErr, delay)
			sess.close() // Kill connection to trigger a fresh dial sequence on next iteration
			if sess.service.sleepContext(ctx, delay) {
				return
			}
		}
	}

	log.Printf("[-] Telemetry record exceeded critical retry budget (%d). Discarding to prevent local pipeline freeze.", MaxRetriesPerEvent)
}

// ============================================================================
// CLEAN SHUTDOWN FLUSH ENGINE
// ============================================================================

func (s *Service) drainQueue(conn net.Conn) {
	s.isDraining.Store(true)

	// Best-effort 5-second deadline to push remaining in-flight memory to disk/wire
	timeout := time.After(5 * time.Second)
	count := 0

	for {
		select {
		case <-timeout:
			log.Printf("[SIEM] Flush cutoff reached. Aborted %d buffered telemetry items.", len(s.eventQueue))
			return
		case event := <-s.eventQueue:
			if conn != nil {
				msg := s.formatSyslog(event)
				conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
				if _, err := conn.Write([]byte(msg + "\n")); err == nil {
					count++
				}
			}
		default:
			log.Printf("[SIEM] Telemetry stream fully synchronized. Flushed %d remaining logs.", count)
			return
		}
	}
}

// ============================================================================
// FORMATTERS & ACCURATE HISTORICAL TIMELINES
// ============================================================================

func (s *Service) formatSyslog(event models.Event) string {
	priority := syslogPriority(event.Severity)
	var t time.Time
	if event.Timestamp == "" {
		t = time.Now()
	} else {
		// Attempt to parse the string (Assuming standard RFC3339 format)
		parsedTime, err := time.Parse(time.RFC3339, event.Timestamp)
		if err != nil {
			// Fallback to now if the string is malformed
			t = time.Now()
		} else {
			t = parsedTime
		}
	}

	timestamp := t.UTC().Format(time.RFC3339Nano)

	detailsJSON, err := json.Marshal(event.Details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	nodeAlias := event.NodeID
	if s.nodeService != nil {
		if nodeDetails, err := s.nodeService.GetNodeDetails(event.NodeID); err == nil && nodeDetails != nil {
			if nodeDetails.Alias != "" {
				nodeAlias = nodeDetails.Alias
			}
		}
	}

	sd := fmt.Sprintf(
		`[honeywire trigger="%s" source="%s" target="%s" node="%s" sensor="%s" severity="%s" details=%s]`,
		event.EventTrigger, event.Source, event.Target,
		nodeAlias, event.SensorID, event.Severity, string(detailsJSON),
	)

	return fmt.Sprintf("<%d>1 %s honeywire honeywireSensor - - %s %s",
		priority, timestamp, sd, event.EventTrigger)
}

func syslogPriority(severity string) int {
	facility := 16 * 8 // local0
	switch severity {
	case "critical":
		return facility + 2
	case "high":
		return facility + 3
	case "medium":
		return facility + 4
	case "low":
		return facility + 5
	default:
		return facility + 6 // info
	}
}

// helper to clean up select-block boilerplate across retry systems
func (s *Service) sleepContext(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	select {
	case <-t.C:
		return false
	case <-ctx.Done():
		t.Stop()
		return true
	}
}

func (s *Service) Wait() {
	s.wg.Wait()
}
