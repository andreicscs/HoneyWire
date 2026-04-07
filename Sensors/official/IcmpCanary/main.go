package main

import (
	"log"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"

	"github.com/honeywire/sdk-go"
)

func main() {
	// 1. Initialize the HoneyWire SDK
	hw := sdk.NewSensor("network")
	hw.Start()

	log.Printf("[*] HoneyWire Ping Canary | Listening for ICMP Echo Requests...")

	// 2. Open a Raw Socket for ICMP (IPv4)
	// Listening on "0.0.0.0" captures traffic on all container interfaces
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		log.Fatalf("[!] FATAL: Failed to listen for ICMP: %v\n(Ensure container has CAP_NET_RAW capability)", err)
	}
	defer conn.Close()

	// 3. The Packet Sniffing Loop
	buf := make([]byte, 1500)
	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			log.Printf("[-] Read error: %v", err)
			continue
		}

		// Parse the raw bytes into an ICMP message (Protocol 1 = ICMP)
		msg, err := icmp.ParseMessage(1, buf[:n])
		if err != nil {
			continue
		}

		// Filter for Echo Requests (Ping)
		if msg.Type == ipv4.ICMPTypeEcho {
			sourceIP := addr.String()
			packetSize := n

			log.Printf("[+] ICMP Echo Request detected from %s (size=%d)", sourceIP, packetSize)

			// 4. Dispatch the Event via the SDK
			hw.ReportEvent(
				"icmp_ping_received", // Event Type
				"high",               // Severity
				map[string]any{       // Details
					"packet_size": packetSize,
				},
				"logged",        // Action Taken
				sourceIP,        // Source
				"ICMP Listener", // Target
			)
		}
	}
}