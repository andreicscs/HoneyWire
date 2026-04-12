package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"github.com/honeywire/sdk-go"
)

func main() {
	hw, err := sdk.NewSensor()
	if err != nil {
		log.Fatalf("[!] FATAL: Failed to initialize sensor: %v", err)
	}

	if hw.TestMode {
		if hw.RunTestMode() {
			log.Println("✅ Test mode complete. Exiting gracefully.")
			os.Exit(0)
		}
		log.Println("❌ Test mode failed to contact Hub.")
		os.Exit(1)
	}

	if err := hw.Start(); err != nil {
		log.Fatalf("[!] FATAL: Failed to sync with Hub: %v", err)
	}
	defer hw.Stop() // Clean up goroutines!

	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		log.Fatalf("[!] FATAL: Failed to listen for ICMP: %v\n(Ensure container has CAP_NET_RAW capability)", err)
	}
	defer conn.Close()

	log.Printf("[*] HoneyWire Ping Canary | Listening for ICMP Echo Requests...")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go listenICMP(conn, hw)

	// Block until shutdown signal
	<-ctx.Done()
	log.Println("[*] Shutdown signal received. Exiting.")
}

func listenICMP(conn *icmp.PacketConn, hw *sdk.Sensor) {
	buf := make([]byte, 1500)
	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			log.Printf("[-] Read error: %v", err)
			continue
		}

		msg, err := icmp.ParseMessage(1, buf[:n])
		if err != nil {
			continue
		}

		if msg.Type != ipv4.ICMPTypeEcho {
			continue
		}

		// Clean the IP string
		sourceIP := addr.String()
		if host, _, err := net.SplitHostPort(sourceIP); err == nil {
			sourceIP = host
		}

		echo, ok := msg.Body.(*icmp.Echo)
		if !ok {
			continue
		}

		log.Printf("[+] ICMP Echo Request from %s (seq=%d size=%d)", sourceIP, echo.Seq, n)

		hw.ReportEvent(
			"high",
			"icmp_ping_received",
			sourceIP,
			"ICMP Listener",
			map[string]any{
				"packet_size":  n,
				"icmp_id":      echo.ID,
				"icmp_seq":     echo.Seq,
				"action_taken": "logged",
			},
		)
	}
}