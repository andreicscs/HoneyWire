package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/honeywire/sdk-go"
)

type ScanState struct {
	history   []hit
	lastAlert time.Time
}

type hit struct {
	timestamp time.Time
	port      uint16
	flags     uint8
	winSize   uint16
}

const maxTrackers = 10000

var (
	threshold   = getEnvInt("HW_SCAN_THRESHOLD", 5)
	window      = time.Duration(getEnvInt("HW_SCAN_WINDOW", 5)) * time.Second
	cooldown    = 60 * time.Second
	ignorePorts = parseIgnorePorts(getEnv("HW_IGNORE_PORTS", "80,443"))

	mu          sync.Mutex
	trackers    = make(map[string]*ScanState)
	trackerList []string
)

func analyzePacket(flags uint8, winSize uint16) (string, string) {
	if flags == 0x2B {
		return "Likely Nmap", "OS Detection Probe (T3)"
	}

	isScannerWindow := (winSize == 1024 || winSize == 2048 || winSize == 4096)

	switch flags {
	case 0x02:
		if isScannerWindow {
			return "Likely Nmap / Masscan", "Stealth SYN Port Scan"
		}
		return "Generic/Evasive Scanner", "SYN Port Probe"

	case 0x00:
		return "Likely Nmap", "Stealth NULL Scan"

	case 0x01:
		return "Likely Nmap", "Stealth FIN Scan"

	case 0x29:
		return "Likely Nmap", "Stealth XMAS Scan"
	}

	return "Generic/Evasive Scanner", "Unidentified Port Probe"
}

func main() {
	hw, err := sdk.NewSensor()
	if err != nil {
		log.Fatalf("[!] FATAL: %v", err)
	}

	hw.SetTestPayload(
		"network_scan_detected",
		"Wizard Firedrill",
		"Multiple Ports",
		sdk.EventDetails{
			{Key: "test_message", Value: "Wizard triggered a synthetic event firedrill."},
			{Key: "ports_hit", Value: []uint16{22, 80, 443, 3306, 8080}},
			{Key: "scan_type", Value: "Stealth SYN Port Scan"},
			{Key: "tool_fingerprint", Value: "Likely Nmap / Masscan"},
			{Key: "count", Value: 5},
			{Key: "window_sec", Value: 5.0},
			{Key: "action_taken", Value: "logged"},
		},
	)

	if hw.TestMode {
		if hw.RunTestMode() {
			os.Exit(0)
		}
		os.Exit(1)
	}

	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP)
	if err != nil {
		log.Fatalf("[!] FATAL: Failed to open raw socket (requires root/CAP_NET_RAW): %v", err)
	}
	defer syscall.Close(fd)

	log.Printf("[*] HoneyWire Scan Detector | Threshold: %d ports | Window: %v", threshold, window)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Garbage Collection Loop
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mu.Lock()
				now := time.Now()
				var activeList []string
				for _, ip := range trackerList {
					state := trackers[ip]
					isActive := false
					if len(state.history) > 0 {
						if now.Sub(state.history[len(state.history)-1].timestamp) <= window {
							isActive = true
						}
					} else {
						if now.Sub(state.lastAlert) <= cooldown {
							isActive = true
						}
					}

					if isActive {
						activeList = append(activeList, ip)
					} else {
						delete(trackers, ip)
					}
				}
				trackerList = activeList
				mu.Unlock()
			}
		}
	}()

	go func() {
		buf := make([]byte, 65536)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, _, err := syscall.Recvfrom(fd, buf, 0)
				if err != nil || n < 20 {
					continue
				}

				ihl := int(buf[0]&0x0F) * 4
				if n < ihl+20 {
					continue
				}

				tcpStart := ihl
				flags := buf[tcpStart+13]

				if flags != 0x02 && flags != 0x2B && flags != 0x00 && flags != 0x01 && flags != 0x29 {
					continue
				}

				srcIP := net.IPv4(buf[12], buf[13], buf[14], buf[15]).String()
				dstPort := (uint16(buf[tcpStart+2]) << 8) | uint16(buf[tcpStart+3])
				winSize := (uint16(buf[tcpStart+14]) << 8) | uint16(buf[tcpStart+15])

				if ignorePorts[dstPort] {
					continue
				}

				processHit(hw, srcIP, dstPort, flags, winSize)
			}
		}
	}()

	if err := hw.Start(); err != nil {
		log.Fatalf("[!] FATAL: %v", err)
	}
	defer hw.Stop()

	<-ctx.Done()
}

func processHit(hw *sdk.Sensor, srcIP string, dstPort uint16, flags uint8, winSize uint16) {
	now := time.Now()

	mu.Lock()
	state, exists := trackers[srcIP]
	if !exists {
		if len(trackers) >= maxTrackers {
			oldest := trackerList[0]
			trackerList = trackerList[1:]
			delete(trackers, oldest)
		}
		state = &ScanState{}
		trackers[srcIP] = state
		trackerList = append(trackerList, srcIP)
	}

	state.history = append(state.history, hit{timestamp: now, port: dstPort, flags: flags, winSize: winSize})

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
	state.history = active

	shouldAlert := false
	if len(uniquePortsList) >= threshold {
		if now.Sub(state.lastAlert) > cooldown {
			state.lastAlert = now
			shouldAlert = true
			state.history = nil
		}
	}
	mu.Unlock()

	if shouldAlert {
		tool := "Generic/Evasive Scanner"
		scanType := "Unidentified Port Probe"

		for _, h := range active {
			t, s := analyzePacket(h.flags, h.winSize)
			if t != "Generic/Evasive Scanner" {
				tool = t
				scanType = s
				break
			}
		}

		log.Printf("[!] Port scan detected from %s: %v | Tool: %s", srcIP, uniquePortsList, tool)

		hw.ReportEvent(
			"network_scan_detected",
			srcIP,
			"Multiple Ports",
			sdk.EventDetails{
				{Key: "ports_hit", Value: uniquePortsList},
				{Key: "scan_type", Value: scanType},
				{Key: "tool_guess", Value: tool},
				{Key: "count", Value: len(uniquePortsList)},
				{Key: "window_sec", Value: window.Seconds()},
				{Key: "action_taken", Value: "logged"},
			},
		)
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
