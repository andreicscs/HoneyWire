package main

import (
	"context"
	"fmt"
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

var (
	decoyPorts  = parsePorts(getEnv("HW_DECOY_PORTS", "2222,3306"))
	tarpitMode  = strings.ToLower(getEnv("HW_TARPIT_MODE", "hold"))
	banner      = parseBanner(getEnv("HW_TARPIT_BANNER", ""))
	concurrency = 1000
	maxBytes    = 50 * 1024
	maxLines    = 10
	maxDuration = 3600 * time.Second
)

func main() {
	hw, err := sdk.NewSensor()
	if err != nil {
		log.Fatalf("[!] FATAL: %v", err)
	}

	if hw.TestMode {
		if hw.RunTestMode() { os.Exit(0) }
		os.Exit(1)
	}

	if err := hw.Start(); err != nil {
		log.Fatalf("[!] FATAL: %v", err)
	}
	defer hw.Stop()

	log.Printf("[*] HoneyWire Tarpit | Mode: %s", strings.ToUpper(tarpitMode))

	semaphore := make(chan struct{}, concurrency)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	for _, port := range decoyPorts {
		go startListener(ctx, hw, port, semaphore)
	}

	<-ctx.Done()
	log.Println("[*] Tarpit shutting down.")
}

func startListener(ctx context.Context, hw *sdk.Sensor, port int, semaphore chan struct{}) {
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	
    var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		log.Printf("[!] Failed to bind to port %d: %v", port, err)
		return
	}
	defer listener.Close()

	log.Printf("[+] Tarpit listening on port %d", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			// Break if context was cancelled
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("[-] Accept error on port %d: %v", port, err)
				continue
			}
		}

		semaphore <- struct{}{}
		go func(c net.Conn) {
			defer func() { <-semaphore }()
			handleConnection(hw, c, port)
		}(conn)
	}
}

func handleConnection(hw *sdk.Sensor, conn net.Conn, port int) {
	defer conn.Close()
	start := time.Now()
	
	remoteAddr := conn.RemoteAddr().String()
	srcIP, _, err := net.SplitHostPort(remoteAddr)
	if err != nil { srcIP = remoteAddr }

	var payload []string
	consumedBytes := 0

	if banner != "" && tarpitMode != "close" {
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		conn.Write([]byte(banner))
	}

	if tarpitMode != "close" {
		buf := make([]byte, 1024)
		for consumedBytes < maxBytes && time.Since(start) < maxDuration {
			conn.SetReadDeadline(time.Now().Add(300 * time.Second))
			n, err := conn.Read(buf)

			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
					conn.Write([]byte{0})
					continue
				}
				break 
			}

			if n > 0 {
				consumedBytes += n
				
				if len(payload) < maxLines {
					text := strings.TrimSpace(strings.ToValidUTF8(string(buf[:n]), "?"))
					if text != "" {
						payload = append(payload, text)
					}
				}

				if tarpitMode == "echo" {
					conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
					conn.Write(buf[:n])
				}

				time.Sleep(500 * time.Millisecond)
			}
		}
	}

	duration := time.Since(start).Seconds()

	hw.ReportEvent(
		"high",
		"tcp_connection",
		srcIP,
		fmt.Sprintf("Port %d", port),
		map[string]any{
			"duration_sec": duration,
			"payload":      payload,
			"action_taken": tarpitMode, 
		},
	)
}

func parsePorts(raw string) []int {
	var ports []int
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p == "" { continue }
		if val, err := strconv.Atoi(p); err == nil {
			ports = append(ports, val)
		}
	}
	if len(ports) == 0 { return []int{2222, 3306} }
	return ports
}

func parseBanner(raw string) string {
	raw = strings.ReplaceAll(raw, "\\r", "\r")
	raw = strings.ReplaceAll(raw, "\\n", "\n")
	return raw
}

func getEnv(key, fallback string) string {
	if val, exists := os.LookupEnv(key); exists { return val }
	return fallback
}