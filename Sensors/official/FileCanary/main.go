package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/honeywire/sdk-go"
)

func main() {
	hw, err := sdk.NewSensor()
	if err != nil {
		log.Fatalf("[!] FATAL: Failed to initialize sensor: %v", err)
	}

	if hw.TestMode {
		if hw.RunTestMode() {
			os.Exit(0)
		}
		os.Exit(1)
	}

	if err := hw.Start(); err != nil {
		log.Fatalf("[!] FATAL: Failed to start SDK: %v", err)
	}
	defer hw.Stop()

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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
			case <-ctx.Done():
				return
			}
		}
	}()

	err = watcher.Add(honeyDir)
	if err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()
	log.Println("[*] Shutting down watcher.")
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