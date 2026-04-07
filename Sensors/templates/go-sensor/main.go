package main

import (
	"log"
	"os"
	"time"

	"github.com/honeywire/sdk-go"
)

var (
	// Load custom configuration from environment variables
	severity = getEnv("HW_SEVERITY", "medium")
	target   = getEnv("HW_CUSTOM_TARGET", "/tmp/honey")
)

func main() {
	// 1. Initialize the HoneyWire SDK
	// Change 'custom' to whatever category fits your sensor (e.g., 'network', 'file', 'auth')
	hw := sdk.NewSensor("custom")

	// Start the SDK (This automatically handles Hub syncing and HW_TEST_MODE for CI/CD)
	hw.Start()

	log.Printf("[*] Starting Custom Go Sensor | Target: %s | Severity: %s", target, severity)

	// 2. --- YOUR SENSOR LOGIC GOES HERE ---
	// This is the main loop of your sensor. Do not let the main function exit!
	for {
		// Example: Waiting for an event...
		time.Sleep(60 * time.Second)

		// Assume an attack happened!
		log.Printf("[!] Attack detected! Gathering forensics...")
		sourceIP := "192.168.1.100" // Replace with actual extracted data

		// 3. Send the alert to the Hub using the SDK's built-in method
		hw.ReportEvent(
			"custom_anomaly_detected", // Event Type
			severity,                  // Severity
			map[string]any{            // Details (Custom JSON payload)
				"attack_type": "example_probe",
				"raw_payload": "GET /etc/passwd HTTP/1.1",
			},
			"logged",      // Action Taken
			sourceIP,      // Source
			"Custom Trap", // Target
		)
	}
}

// Helper function to easily grab environment variables with fallbacks
func getEnv(key, fallback string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return fallback
}