package siem

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/honeywire/hub/internal/models"
)

var EventQueue = make(chan models.Event, 5000)

type SiemConfig struct {
	Address  string // "host:port"
	Protocol string // "tcp" or "udp"
}

var ForwarderConfig = SiemConfig{
	Address:  "",
	Protocol: "tcp",
}

// If the queue is full, new events are dropped with a log message.
func StartWorker() {
	go func() {
		for event := range EventQueue {
			forwardSyslog(event)
		}
	}()
	log.Println("[SIEM] Worker started. Listening for events...")
}

func forwardSyslog(event models.Event) {
	if ForwarderConfig.Address == "" {
		return
	}

	priority := syslogPriority(event.Severity)
	timestamp := time.Now().Format("Jan 02 15:04:05")
	hostname := "honeywire"
	tag := "honeywire-sensor"

	// serialize the Details map into a JSON string
	detailsJSON, err := json.Marshal(event.Details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	msg := fmt.Sprintf("<%d>%s %s %s[%d]: [%s] Trigger: %s | Source: %s | Target: %s | Sensor: %s | Details: %s",
		priority,
		timestamp,
		hostname,
		tag,
		event.ID,
		event.Severity,
		event.EventTrigger,
		event.Source,
		event.Target,
		event.SensorID,
		string(detailsJSON),
	)

	switch ForwarderConfig.Protocol {
	case "tcp":
		forwardTCP(msg)
	case "udp":
		forwardUDP(msg)
	default:
		log.Printf("[!] Unknown SIEM protocol: %s", ForwarderConfig.Protocol)
	}
}

func forwardTCP(message string) {
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.Dial("tcp", ForwarderConfig.Address)
	if err != nil {
		log.Printf("[!] SIEM TCP connection failed: %v", err)
		return
	}
	defer conn.Close()

	// Set write deadline to prevent hanging if the firewall drops packets.
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

	_, err = conn.Write([]byte(message + "\n"))
	if err != nil {
		log.Printf("[!] SIEM TCP write failed: %v", err)
	}
}

func forwardUDP(message string) {
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.Dial("udp", ForwarderConfig.Address)
	if err != nil {
		log.Printf("[!] SIEM UDP connection failed: %v", err)
		return
	}
	defer conn.Close()

	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

	_, err = conn.Write([]byte(message + "\n"))
	if err != nil {
		log.Printf("[!] SIEM UDP write failed: %v", err)
	}
}

// syslogPriority converts severity to syslog priority level.
// Syslog priority = Facility * 8 + Severity (using local0 facility = 16)
func syslogPriority(severity string) int {
	facility := 16 * 8 // local0
	severityMap := map[string]int{
		"critical": 2, // crit
		"high":     3, // err
		"medium":   4, // warning
		"low":      5, // notice
		"info":     6, // info
	}
	level, exists := severityMap[severity]
	if !exists {
		level = 6 // default to info
	}
	return facility + level
}

func FlushQueue(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		select {
		case event := <-EventQueue:
			forwardSyslog(event)
		case <-time.After(100 * time.Millisecond):
			// Check if queue is truly empty by trying a non-blocking read.
			select {
			case event := <-EventQueue:
				forwardSyslog(event)
			default:
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("SIEM flush timeout exceeded")
		}
	}
}

func UpdateConfig(address, protocol string) {
	if protocol == "" {
		protocol = "tcp"
	}
	ForwarderConfig.Address = address
	ForwarderConfig.Protocol = protocol
}
