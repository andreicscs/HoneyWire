package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/honeywire/hub/internal/api"
	"github.com/honeywire/hub/internal/auth"
	"github.com/honeywire/hub/internal/config"
	"github.com/honeywire/hub/internal/notify"
	"github.com/honeywire/hub/internal/siem"
	"github.com/honeywire/hub/internal/store"
)

// We will eventually load this from internal/config
const Version = "1.1.0"

func main() {
	log.Println("Starting HoneyWire Go Hub initialization...")

	cfg := config.Load()

	dbStore, err := store.NewStore(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer dbStore.DB.Close()

	sessionStore := auth.NewSessionStore()

	r := api.SetupRouter(cfg, dbStore, sessionStore)

	notify.StartWorker()
	siem.StartWorker()

	// Load initial configurations from DB
	var isArmed, webhookType, webhookURL, webhookEvents string
	dbStore.DB.QueryRow("SELECT value FROM config WHERE key = 'is_armed'").Scan(&isArmed)
	dbStore.DB.QueryRow("SELECT value FROM config WHERE key = 'webhook_type'").Scan(&webhookType)
	dbStore.DB.QueryRow("SELECT value FROM config WHERE key = 'webhook_url'").Scan(&webhookURL)
	dbStore.DB.QueryRow("SELECT value FROM config WHERE key = 'webhook_events'").Scan(&webhookEvents)
	notify.UpdateConfig(isArmed == "true", webhookType, webhookURL, webhookEvents)

	var siemAddress, siemProtocol string
	dbStore.DB.QueryRow("SELECT value FROM config WHERE key = 'siem_address'").Scan(&siemAddress)
	dbStore.DB.QueryRow("SELECT value FROM config WHERE key = 'siem_protocol'").Scan(&siemProtocol)

	if siemProtocol == "" {
		siemProtocol = "tcp" // default
	}
	siem.UpdateConfig(siemAddress, siemProtocol)

	if siemAddress != "" {
		log.Printf("[SIEM] Configured to forward to %s via %s\n", siemAddress, siemProtocol)
	} else {
		log.Println("[SIEM] Forwarding disabled (no address configured).")
	}

	// Start the HTTP server in a goroutine
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("Server listening on port %s\n", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	// Block until we receive a shutdown signal
	<-sigChan
	log.Println("\n[*] Received shutdown signal. Initiating graceful shutdown...")

	// Create a context with a 10-second timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop accepting new HTTP connections
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("[!] Server forced to shut down: %v\n", err)
	}

	log.Println("[*] Flushing pending webhooks...")
	if err := notify.FlushQueue(5 * time.Second); err != nil {
		log.Printf("[!] Webhook flush error: %v\n", err)
	}

	log.Println("[*] Flushing pending SIEM events...")
	if err := siem.FlushQueue(5 * time.Second); err != nil {
		log.Printf("[!] SIEM flush error: %v\n", err)
	}

	log.Println("[*] Graceful shutdown complete. Goodbye!")
}
