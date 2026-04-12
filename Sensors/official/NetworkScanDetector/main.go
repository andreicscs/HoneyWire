package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
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
}

var (
	threshold   = getEnvInt("HW_SCAN_THRESHOLD", 5)
	window      = time.Duration(getEnvInt("HW_SCAN_WINDOW", 5)) * time.Second
	cooldown    = 60 * time.Second
	ignorePorts = parseIgnorePorts(getEnv("HW_IGNORE_PORTS", "80,443"))
	trackers    = make(map[string]*ScanState)
)

func main() {
	hw, err := sdk.NewSensor()
	if err != nil {
		log.Fatalf("[!] FATAL: %v", err)
	}

	if hw.TestMode {
		if hw.RunTestMode() {
			os.Exit(0)
		}
		os.Exit(1)
	}

	if err := hw.Start(); err != nil {
		log.Fatalf("[!] FATAL: %v", err)
	}
	defer hw.Stop()

	log.Printf("[*] HoneyWire Scan Detector | Threshold: %d ports | Window: %v", threshold, window)

	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP)
	if err != nil {
		log.Fatalf("[!] FATAL: Failed to open raw socket (requires root/CAP_NET_RAW): %v", err)
	}
	defer syscall.Close(fd)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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

				if flags != 0x02 { // SYN
					continue
				}

				srcIP := net.IPv4(buf[12], buf[13], buf[14], buf[15]).String()
				dstPort := (uint16(buf[tcpStart+2]) << 8) | uint16(buf[tcpStart+3])

				if ignorePorts[dstPort] {
					continue
				}

				processHit(hw, srcIP, dstPort)
			}
		}
	}()

	<-ctx.Done()
}

func processHit(hw *sdk.Sensor, srcIP string, dstPort uint16) {
	now := time.Now()

	state, exists := trackers[srcIP]
	if !exists {
		state = &ScanState{}
		trackers[srcIP] = state
	}

	state.history = append(state.history, hit{timestamp: now, port: dstPort})

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
			state.history = nil
		}
	}
}

func parseIgnorePorts(raw string) map[uint16]bool {
	ports := make(map[uint16]bool)
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p == "" { continue }
		val, err := strconv.ParseUint(p, 10, 16)
		if err == nil { ports[uint16(val)] = true }
	}
	return ports
}

func getEnv(key, fallback string) string {
	if val, exists := os.LookupEnv(key); exists { return val }
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(val); err == nil { return intVal }
	}
	return fallback
}