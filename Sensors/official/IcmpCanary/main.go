package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/honeywire/sdk-go"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func main() {
	hw, err := sdk.NewSensor()
	if err != nil {
		log.Fatalf("[!] FATAL: Failed to initialize sensor: %v", err)
	}

	hw.SetTestPayload(
		"icmp_scan_received",
		"Wizard Firedrill",
		"ICMP Listener",
		sdk.EventDetails{
			{Key: "test_message", Value: "Wizard triggered a synthetic event firedrill."},
			{Key: "icmp_type", Value: "Timestamp Request"},
			{Key: "packet_size", Value: 64},
			{Key: "icmp_id", Value: 1337},
			{Key: "icmp_seq", Value: 1},
		},
	)

	if hw.TestMode {
		if hw.RunTestMode() {
			log.Println("✅ Test mode complete. Exiting gracefully.")
			os.Exit(0)
		}
		log.Println("❌ Test mode failed to contact Hub.")
		os.Exit(1)
	}

	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		log.Fatalf("[!] FATAL: Failed to listen for ICMP: %v\n(Ensure container has CAP_NET_RAW capability)", err)
	}
	defer conn.Close()

	log.Printf("[*] HoneyWire Ping Canary | Listening for ICMP Echo Requests...")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go listenICMP(conn, hw)

	if err := hw.Start(); err != nil {
		log.Fatalf("[!] FATAL: Failed to sync with Hub: %v", err)
	}
	defer hw.Stop() // Clean up goroutines!

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

		// Clean the IP string
		sourceIP := addr.String()
		if host, _, err := net.SplitHostPort(sourceIP); err == nil {
			sourceIP = host
		}

		var icmpTypeStr string
		var icmpID, icmpSeq int

		if msg.Type == ipv4.ICMPTypeEcho {
			icmpTypeStr = "Echo Request (Ping)"
			if echo, ok := msg.Body.(*icmp.Echo); ok {
				icmpID = echo.ID
				icmpSeq = echo.Seq
			}
		} else if msg.Type == ipv4.ICMPTypeTimestamp {
			icmpTypeStr = "Timestamp Request"
			// Type 13 usually carries 12 bytes of data (ID, Seq, Timestamps).
			// x/net/icmp parses it into a RawBody
			if raw, ok := msg.Body.(*icmp.RawBody); ok && len(raw.Data) >= 4 {
				icmpID = int(raw.Data[0])<<8 | int(raw.Data[1])
				icmpSeq = int(raw.Data[2])<<8 | int(raw.Data[3])
			}
		} else {
			continue // We only care about Ping and Timestamp scans
		}

		log.Printf("[+] ICMP %s from %s (seq=%d size=%d)", icmpTypeStr, sourceIP, icmpSeq, n)

		hw.ReportEvent(
			"icmp_scan_received",
			sourceIP,
			"ICMP Listener",
			sdk.EventDetails{
				{Key: "icmp_type", Value: icmpTypeStr},
				{Key: "packet_size", Value: n},
				{Key: "icmp_id", Value: icmpID},
				{Key: "icmp_seq", Value: icmpSeq},
			},
		)
	}
}
