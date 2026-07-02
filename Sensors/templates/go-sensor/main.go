package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/honeywire/sdk-go"
)

var (
	// Load custom configuration from environment variables
	target   = getEnv("HW_CUSTOM_TARGET", "/tmp/honey")
)

func main() {
	// 1. Initialize the HoneyWire SDK safely
	hw, err := sdk.NewSensor()
	if err != nil {
		log.Fatalf("[!] FATAL: %v", err)
	}

	hw.SetTestPayload(
		"custom_anomaly_detected",
		"Wizard Firedrill",
		"Mock Custom Target",
		sdk.EventDetails{
			{Key: "test_message", Value: "Wizard triggered a synthetic event firedrill."},
			{Key: "action_taken", Value: "logged"},
		},
	)

	// 2. Handle CI/CD Test Mode
	if hw.TestMode {
		if hw.RunTestMode() {
			os.Exit(0)
		}
		os.Exit(1)
	}

	log.Printf("[*] Starting Custom Go Sensor | Target: %s", target)

	// 3. Setup graceful shutdown and acquire resources (e.g., net.Listen)
	// IMPORTANT: Always acquire resources and start your background listeners BEFORE calling hw.Start().
	// This prevents the sensor from reporting a false "online" state to the Hub if it crashes immediately (e.g., port already in use).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Spin off your sensor logic into a non-blocking goroutine
	go runSensor(ctx, hw)

	// 4. Start the SDK (Syncs Hub version, starts heartbeats)
	if err := hw.Start(); err != nil {
		log.Fatalf("[!] FATAL: %v", err)
	}
	defer hw.Stop() // Cleans up the background heartbeat goroutine

	// Block main thread until Ctrl+C is pressed
	<-ctx.Done()
	log.Println("[*] Shutting down Custom Sensor...")
}

func runSensor(ctx context.Context, hw *sdk.Sensor) {
	// Use a Ticker instead of time.Sleep so the loop can be interrupted cleanly
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return // Exit cleanly if OS signal is received

		case <-ticker.C:
			// --- YOUR SENSOR LOGIC GOES HERE ---
			// Assume an attack happened!
			log.Printf("[!] Attack detected! Gathering forensics...")
			sourceIP := "192.168.1.100" // Replace with actual extracted data

			// Send the alert to the Hub using the SDK's built-in method
			hw.ReportEvent(
				"custom_anomaly_detected", // 1. Event Trigger
				sourceIP,                  // 2. Source
				target,                    // 3. Target
				sdk.EventDetails{ // 4. Details
					{Key: "attack_type", Value: "example_probe"},
					{Key: "raw_payload", Value: "GET /etc/passwd HTTP/1.1"},
					{Key: "action_taken", Value: "logged"},
				},
			)
		}
	}
}

// Helper function to easily grab environment variables with fallbacks
func getEnv(key, fallback string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return fallback
}