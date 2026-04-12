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
	severity = getEnv("HW_SEVERITY", "medium")
	target   = getEnv("HW_CUSTOM_TARGET", "/tmp/honey")
)

func main() {
	// 1. Initialize the HoneyWire SDK safely
	hw, err := sdk.NewSensor()
	if err != nil {
		log.Fatalf("[!] FATAL: %v", err)
	}

	// 2. Handle CI/CD Test Mode
	if hw.TestMode {
		if hw.RunTestMode() {
			os.Exit(0)
		}
		os.Exit(1)
	}

	// 3. Start the SDK (Syncs Hub version, starts heartbeats)
	if err := hw.Start(); err != nil {
		log.Fatalf("[!] FATAL: %v", err)
	}
	defer hw.Stop() // Cleans up the background heartbeat goroutine

	log.Printf("[*] Starting Custom Go Sensor | Target: %s | Severity: %s", target, severity)

	// 4. Setup graceful shutdown via OS signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Spin off your sensor logic into a non-blocking goroutine
	go runSensor(ctx, hw)

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
				severity,                  // 1. Severity
				"custom_anomaly_detected", // 2. Event Trigger
				sourceIP,                  // 3. Source
				target,                    // 4. Target
				map[string]any{            // 5. Details
					"attack_type":  "example_probe",
					"raw_payload":  "GET /etc/passwd HTTP/1.1",
					"action_taken": "logged",
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