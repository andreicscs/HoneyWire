package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/honeywire/sdk-go"
)

var (
	decoyPorts  = parsePorts(getEnv("HW_DECOY_PORTS", "2222,3306"))
	tarpitMode  = strings.ToLower(getEnv("HW_TARPIT_MODE", "hold"))
	banner      = parseBanner(getEnv("HW_TARPIT_BANNER", ""))
	concurrency = 1000
	maxBytes    = 50 * 1024 // 50KB
	maxLines    = 10
	maxDuration = 3600 * time.Second
)

func main() {
	// 1. Initialize SDK
	hw := sdk.NewSensor("tarpit")
	hw.Start()

	log.Printf("[*] HoneyWire Tarpit | Mode: %s", strings.ToUpper(tarpitMode))
	if banner != "" {
		log.Printf("[*] Banner loaded: %s", strconv.Quote(banner))
	}

	// 2. Concurrency Limiter (Semaphore)
	semaphore := make(chan struct{}, concurrency)
	errCh := make(chan error)

	// 3. Start a TCP Listener for each Decoy Port
	for _, port := range decoyPorts {
		go startListener(hw, port, semaphore, errCh)
	}

	// Block main thread and watch for fatal listener errors
	log.Fatal(<-errCh)
}

func startListener(hw *sdk.Sensor, port int, semaphore chan struct{}, errCh chan error) {
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		errCh <- fmt.Errorf("[!] FATAL: Failed to bind to port %d: %v", port, err)
		return
	}
	defer listener.Close()

	log.Printf("[+] Tarpit listening on port %d", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("[-] Accept error on port %d: %v", port, err)
			continue
		}

		// Acquire semaphore slot (blocks if 1000 connections are active)
		semaphore <- struct{}{}

		// Spawn a lightweight goroutine for the attacker
		go func(c net.Conn) {
			defer func() { <-semaphore }() // Release slot when done
			handleConnection(hw, c, port)
		}(conn)
	}
}

func handleConnection(hw *sdk.Sensor, conn net.Conn, port int) {
	defer conn.Close()

	start := time.Now()
	
	// Extract the IP without the attacker's ephemeral port
	remoteAddr := conn.RemoteAddr().String()
	srcIP, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		srcIP = remoteAddr
	}

	var payload []string
	consumedBytes := 0

	// 1. Send Fake Banner (e.g., pretending to be SSH)
	if banner != "" && tarpitMode != "close" {
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		conn.Write([]byte(banner))
	}

	// 2. The Tarpit Trap Loop
	if tarpitMode != "close" {
		buf := make([]byte, 1024)
		for consumedBytes < maxBytes && time.Since(start) < maxDuration {
			// Wait up to 5 minutes for the attacker to send data
			conn.SetReadDeadline(time.Now().Add(300 * time.Second))
			n, err := conn.Read(buf)

			if err != nil {
				// If we timed out waiting for them, send a Null byte to keep the connection alive!
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
					conn.Write([]byte{0})
					continue
				}
				break // Attacker dropped the connection or sent EOF
			}

			if n > 0 {
				consumedBytes += n
				
				// Safely decode to string and log up to maxLines
				if len(payload) < maxLines {
					text := strings.TrimSpace(strings.ToValidUTF8(string(buf[:n]), "?"))
					if text != "" {
						payload = append(payload, text)
					}
				}

				// If in echo mode, reflect their attack back at them
				if tarpitMode == "echo" {
					conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
					conn.Write(buf[:n])
				}

				// The Tar: Slow down our processing to waste their time
				time.Sleep(500 * time.Millisecond)
			}
		}
	}

	duration := time.Since(start).Seconds()

	// 3. Dispatch the Event via the SDK
	hw.ReportEvent(
		"tcp_connection",
		"high",
		map[string]any{
			"duration_sec": duration,
			"payload":      payload,
		},
		tarpitMode,
		srcIP,
		fmt.Sprintf("Port %d", port),
	)
}

// --- Helper Functions ---

func parsePorts(raw string) []int {
	var ports []int
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p == "" { continue }
		if val, err := strconv.Atoi(p); err == nil {
			ports = append(ports, val)
		} else {
			log.Printf("[!] Invalid port in HW_DECOY_PORTS: %s", p)
		}
	}
	if len(ports) == 0 {
		return []int{2222, 3306}
	}
	return ports
}

// Replaces "\n" string literals from env vars into actual newlines
func parseBanner(raw string) string {
	raw = strings.ReplaceAll(raw, "\\r", "\r")
	raw = strings.ReplaceAll(raw, "\\n", "\n")
	return raw
}

func getEnv(key, fallback string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return fallback
}