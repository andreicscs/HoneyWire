package siem

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/honeywire/hub/internal/models"
)

type Service struct {
	eventQueue chan models.Event
	address    string
	protocol   string
	mu         sync.RWMutex
}

func NewService() *Service {
	return &Service{
		eventQueue: make(chan models.Event, 5000),
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
	select {
	case s.eventQueue <- event:
	default:
		log.Println("[!] SIEM Queue full, dropping event")
	}
}

func (s *Service) StartWorker(ctx context.Context) {
	log.Println("[SIEM] Worker started. Listening for events...")
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("[SIEM] Worker stopped.")
				return
			case event := <-s.eventQueue:
				s.forwardSyslog(event)
			}
		}
	}()
}

func (s *Service) forwardSyslog(event models.Event) {
	s.mu.RLock()
	address := s.address
	protocol := s.protocol
	s.mu.RUnlock()

	if address == "" {
		return
	}

	priority := syslogPriority(event.Severity)
	timestamp := time.Now().Format("Jan 02 15:04:05")
	hostname := "honeywire"
	tag := "honeywire-sensor"

	detailsJSON, err := json.Marshal(event.Details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	msg := fmt.Sprintf("<%d>%s %s %s[%d]: [%s] Trigger: %s | Source: %s | Target: %s | Sensor: %s | Details: %s",
		priority, timestamp, hostname, tag, event.ID, event.Severity,
		event.EventTrigger, event.Source, event.Target, event.SensorID, string(detailsJSON))

	switch protocol {
	case "tcp":
		s.forwardTCP(address, msg)
	case "udp":
		s.forwardUDP(address, msg)
	default:
		log.Printf("[!] Unknown SIEM protocol: %s", protocol)
	}
}

func (s *Service) forwardTCP(address, message string) {
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.Dial("tcp", address)
	if err != nil {
		log.Printf("[!] SIEM TCP connection failed: %v", err)
		return
	}
	defer conn.Close()
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write([]byte(message + "\n")); err != nil {
		log.Printf("[!] SIEM TCP write failed: %v", err)
	}
}

func (s *Service) forwardUDP(address, message string) {
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.Dial("udp", address)
	if err != nil {
		log.Printf("[!] SIEM UDP connection failed: %v", err)
		return
	}
	defer conn.Close()
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write([]byte(message + "\n")); err != nil {
		log.Printf("[!] SIEM UDP write failed: %v", err)
	}
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

func (s *Service) FlushQueue(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for len(s.eventQueue) > 0 {
		if time.Now().After(deadline) {
			return fmt.Errorf("SIEM flush timeout exceeded, %d events remaining", len(s.eventQueue))
		}
		event := <-s.eventQueue
		s.forwardSyslog(event)
	}
	return nil
}
