package main

import (
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/honeywire/sdk-go"
)

// ScanState tracks the history and cooldown for a single IP
type ScanState struct {
	history   []hit
	lastAlert time.Time
}

type hit struct {
	timestamp time.Time
	port      uint16
}

var (
	threshold   = getEnvInt("HW_SCAN_THRESHOLD", 5)
	window      = time.Duration(getEnvInt("HW_SCAN_WINDOW", 5)) * time.Second
	cooldown    = 60 * time.Second
	ignorePorts = parseIgnorePorts(getEnv("HW_IGNORE_PORTS", "80,443"))
	trackers    = make(map[string]*ScanState)
)

func main() {
	// 1. Initialize SDK
	hw := sdk.NewSensor()
	hw.Start()

	log.Printf("[*] HoneyWire Scan Detector | Threshold: %d ports | Window: %v", threshold, window)

	// 2. Open Raw Socket (Captures all TCP packets at the OS level)
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP)
	if err != nil {
		log.Fatalf("[!] FATAL: Failed to open raw socket (requires root/CAP_NET_RAW): %v", err)
	}
	defer syscall.Close(fd)

	// 3. Packet Sniffing Loop
	buf := make([]byte, 65536)
	for {
		n, _, err := syscall.Recvfrom(fd, buf, 0)
		if err != nil || n < 20 {
			continue
		}

		// Calculate IP Header Length (IHL is the lower 4 bits of byte 0, multiplied by 4 bytes)
		ihl := int(buf[0]&0x0F) * 4
		if n < ihl+20 {
			continue // Packet too small to contain a TCP header
		}

		// Navigate to the TCP Header
		tcpStart := ihl
		flags := buf[tcpStart+13]

		// STRICT FILTER: 0x02 is pure SYN. Drop ACKs, SYN-ACKs, FINs, etc.
		if flags != 0x02 {
			continue
		}

		// Extract Source IP and Destination Port
		srcIP := net.IPv4(buf[12], buf[13], buf[14], buf[15]).String()
		dstPort := (uint16(buf[tcpStart+2]) << 8) | uint16(buf[tcpStart+3])

		// Skip ignored ports
		if ignorePorts[dstPort] {
			continue
		}

		processHit(hw, srcIP, dstPort)
	}
}

func processHit(hw *sdk.Sensor, srcIP string, dstPort uint16) {
	now := time.Now()

	state, exists := trackers[srcIP]
	if !exists {
		state = &ScanState{}
		trackers[srcIP] = state
	}

	// Add the new hit
	state.history = append(state.history, hit{timestamp: now, port: dstPort})

	// Cleanup old hits and count unique ports
	var active []hit
	uniquePortsMap := make(map[uint16]bool)
	var uniquePortsList []uint16

	for _, h := range state.history {
		if now.Sub(h.timestamp) <= window {
			active = append(active, h)
			if !uniquePortsMap[h.port] {
				uniquePortsMap[h.port] = true
				uniquePortsList = append(uniquePortsList, h.port)
			}
		}
	}
	state.history = active // Save the cleaned up history

	// Check threshold
	if len(uniquePortsList) >= threshold {
		if now.Sub(state.lastAlert) > cooldown {
			state.lastAlert = now
			
			log.Printf("[!] Port scan detected from %s: %v", srcIP, uniquePortsList)

			hw.ReportEvent(
				"high",
				"network_scan_detected",
				srcIP,
				"Multiple Ports",
				map[string]any{
					"ports_hit":    uniquePortsList,
					"count":        len(uniquePortsList),
					"window_sec":   window.Seconds(),
					"action_taken": "logged",
				},
			)
			
			// Clear queue to save memory after an alert
			state.history = nil
		}
	}
}

func parseIgnorePorts(raw string) map[uint16]bool {
	ports := make(map[uint16]bool)
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		val, err := strconv.ParseUint(p, 10, 16)
		if err == nil {
			ports[uint16(val)] = true
		} else {
			log.Printf("[!] Invalid port in HW_IGNORE_PORTS: %s", p)
		}
	}
	return ports
}

func getEnv(key, fallback string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return fallback
}