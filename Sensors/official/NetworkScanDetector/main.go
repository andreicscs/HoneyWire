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
	history    []hit
	lastAlerts map[string]time.Time
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

func htons(val uint16) uint16 {
	return (val << 8) | (val >> 8)
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
			{Key: "ports_hit", Value: []string{"22", "80", "443", "3306", "8080", "..."}},
			{Key: "count", Value: 1000},
			{Key: "scan_type", Value: "Stealth SYN Port Scan"},
			{Key: "tool_guess", Value: "Likely Nmap / Masscan"},
			{Key: "window_sec", Value: 5.0},
		},
	)

	if hw.TestMode {
		if hw.RunTestMode() {
			os.Exit(0)
		}
		os.Exit(1)
	}

	// AF_PACKET + SOCK_DGRAM bypasses host firewalls (iptables) that drop invalid TCP packets.
	// ETH_P_IP (0x0800) filters for IPv4, and SOCK_DGRAM automatically strips the Ethernet/VLAN link headers
	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_DGRAM, int(htons(0x0800)))
	if err != nil {
		log.Fatalf("[!] FATAL: Failed to open packet socket (requires root/CAP_NET_RAW): %v", err)
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
					}
					if !isActive {
						for _, lastAlert := range state.lastAlerts {
							if now.Sub(lastAlert) <= cooldown {
								isActive = true
								break
							}
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

	ignoreLocal := getEnv("HW_IGNORE_LOCALHOST", "true") == "true"
	loopbackIfIndex := -1
	if lo, err := net.InterfaceByName("lo"); err == nil {
		loopbackIfIndex = lo.Index
	}

	go func() {
		buf := make([]byte, 65536)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, from, err := syscall.Recvfrom(fd, buf, 0)
				if err != nil || n < 20 {
					continue
				}

				// Drop packets from the loopback interface if ignoreLocal is true
				if ignoreLocal && loopbackIfIndex != -1 {
					if ll, ok := from.(*syscall.SockaddrLinklayer); ok && ll.Ifindex == loopbackIfIndex {
						continue
					}
				}

				// Must be TCP (Protocol 6)
				if buf[9] != 0x06 {
					continue
				}

				ihl := int(buf[0]&0x0F) * 4
				if n < ihl+20 {
					continue
				}

				tcpStart := ihl
				// Extract flags and strip ECN/CWR bits (0x3F = 00111111) to ensure clean fingerprinting
				flags := buf[tcpStart+13] & 0x3F

				if flags != 0x02 && flags != 0x2B && flags != 0x00 && flags != 0x01 && flags != 0x29 {
					continue
				}

				srcIP := net.IPv4(buf[12], buf[13], buf[14], buf[15]).String()
				dstPort := (uint16(buf[tcpStart+2]) << 8) | uint16(buf[tcpStart+3])
				winSize := (uint16(buf[tcpStart+14]) << 8) | uint16(buf[tcpStart+15])

				// only ignore legitimate ports (80, 443) for standard SYN scans to prevent noise.
				// Highly anomalous packets (FIN, NULL, XMAS, OS Probes) are inherently malicious
				// and should not be ignored, even if they target an ignored port.
				if flags == 0x02 && ignorePorts[dstPort] {
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
		state = &ScanState{
			lastAlerts: make(map[string]time.Time),
		}
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

	// Determine the highest fidelity signature in the active window
	tool := "Generic/Evasive Scanner"
	scanType := "Unidentified Port Probe"
	isInstant := false

	// Give OS Detection the highest precedence to prevent being masked by NULL/XMAS probes
	hasOSProbe := false

	for _, h := range active {
		t, s := analyzePacket(h.flags, h.winSize)
		if t != "Generic/Evasive Scanner" {
			tool = t
			scanType = s
		}
		if h.flags == 0x2B {
			hasOSProbe = true
			isInstant = true
		} else if !hasOSProbe && (h.flags == 0x00 || h.flags == 0x29 || h.flags == 0x01) {
			isInstant = true
		}
	}

	if hasOSProbe {
		tool, scanType = analyzePacket(0x2B, 0)
	}

	shouldAlert := false
	if isInstant || len(uniquePortsList) >= threshold {
		if now.Sub(state.lastAlerts[scanType]) > cooldown {
			state.lastAlerts[scanType] = now
			shouldAlert = true
			if !isInstant {
				state.history = nil
			}
		}
	}
	mu.Unlock()

	if shouldAlert {
		var displayPorts []string
		for i, p := range uniquePortsList {
			if i >= 5 {
				displayPorts = append(displayPorts, "...")
				break
			}
			displayPorts = append(displayPorts, strconv.Itoa(int(p)))
		}

		log.Printf("[!] %s detected from %s: %v | Tool: %s", scanType, srcIP, displayPorts, tool)

		hw.ReportEvent(
			"network_scan_detected",
			srcIP,
			"Multiple Ports",
			sdk.EventDetails{
				{Key: "ports_hit", Value: displayPorts},
				{Key: "count", Value: len(uniquePortsList)},
				{Key: "scan_type", Value: scanType},
				{Key: "tool_guess", Value: tool},
				{Key: "window_sec", Value: window.Seconds()},
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
