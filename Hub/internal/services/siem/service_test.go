package siem

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"regexp"
	"testing"
	"time"

	"github.com/honeywire/hub/internal/models"
)

func TestSyslogForwardingRFC5424(t *testing.T) {
	// Start a local TCP listener to act as the SIEM server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	address := listener.Addr().String()

	// Initialize the SIEM service
	svc := NewService(nil)
	svc.UpdateConfig(address, "tcp")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc.StartWorker(ctx)

	// Create a test event
	eventTime := time.Date(2023, time.August, 7, 15, 4, 5, 0, time.UTC)
	event := models.Event{
		ID:           123,
		NodeID:       "node-01",
		Timestamp:    eventTime.Format(time.RFC3339),
		Severity:     "critical",
		EventTrigger: "File access",
		Source:       "192.168.1.100",
		Target:       "10.0.0.5",
		SensorID:     "sensor-01",
		Details:      json.RawMessage(`{"file": "/etc/shadow"}`),
	}

	// Send the event
	svc.QueueEvent(event)

	// Accept the connection from the service
	conn, err := listener.Accept()
	if err != nil {
		t.Fatalf("Failed to accept connection: %v", err)
	}
	defer conn.Close()

	// Read the forwarded syslog message
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	reader := bufio.NewReader(conn)
	msg, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read from connection: %v", err)
	}

	// Output the received message
	t.Logf("Received syslog message: %s", msg)

	// RFC5424 Regex Check
	// <PRIVAL>VERSION TIMESTAMP HOSTNAME APP-NAME PROCID MSGID STRUCTURED-DATA MSG
	rfc5424Regex := `^<(\d+)>([1-9]\d{0,2})\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\[.*?\]|-)(?:\s+(.*))?\n$`
	re := regexp.MustCompile(rfc5424Regex)
	
	if !re.MatchString(msg) {
		t.Errorf("Message does not strictly match RFC5424.\nMessage: %q\nRegex: %s", msg, rfc5424Regex)
	} else {
		matches := re.FindStringSubmatch(msg)
		t.Logf("Matches: PRI=%s, VER=%s, TS=%q, HOST=%s, APP=%s, PROC=%s, MSGID=%s, SD=%s, MSG=%s", 
			matches[1], matches[2], matches[3], matches[4], matches[5], matches[6], matches[7], matches[8], matches[9])
	}
}
