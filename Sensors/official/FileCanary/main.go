package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/honeywire/sdk-go"
)

func main() {
	// 1. Initialize the HoneyWire SDK
	hw := sdk.NewSensor()
	hw.Start() // Syncs version, starts heartbeats, runs tests

	// 2. Setup Sensor-specific logic
	honeyDir := getEnv("HW_HONEY_DIR", "/honey_dir")
	if _, err := os.Stat(honeyDir); os.IsNotExist(err) {
		log.Fatalf("[!] FATAL: Watch directory does not exist: %s", honeyDir)
	}

	log.Printf("[*] HoneyWire File Canary | Watching %s", honeyDir)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// 3. The Monitor Loop
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				handleFSEvent(hw, event)
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("[!] Watcher error: %v", err)
			}
		}
	}()

	err = watcher.Add(honeyDir)
	if err != nil {
		log.Fatal(err)
	}

	// Block main thread forever
	<-make(chan struct{})
}

func handleFSEvent(hw *sdk.Sensor, event fsnotify.Event) {
	var action string

	if event.Has(fsnotify.Write) {
		action = "File Modified/Encrypted"
	} else if event.Has(fsnotify.Remove) {
		action = "File Deleted"
	} else if event.Has(fsnotify.Rename) {
		action = "File Moved/Renamed"
	} else {
		return
	}

	// 4. Use the SDK to dispatch the event (no manual HTTP building required!)
	hw.ReportEvent(
		"critical",
		"file_tampered",
		"Unknown (Local OS)",
		event.Name,
		map[string]any{
			"action":       action,
			"action_taken": "logged",
		},
	)
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}