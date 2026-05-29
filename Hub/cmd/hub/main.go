package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/honeywire/hub/internal/api"
	"github.com/honeywire/hub/internal/services/auth"
	"github.com/honeywire/hub/internal/services/config"
	"github.com/honeywire/hub/internal/services/event"
	"github.com/honeywire/hub/internal/services/node"
	"github.com/honeywire/hub/internal/services/notify"
	"github.com/honeywire/hub/internal/services/sensor"
	"github.com/honeywire/hub/internal/services/siem"
	"github.com/honeywire/hub/internal/services/websocket"
	"github.com/honeywire/hub/internal/store"
)

const Version = "2.0.0"

// loadConfigSafe is a helper to fetch DB configs without panicking on empty/missing rows
func loadConfigSafe(s *store.SQLiteStore, key string, fallback string) string {
	val, err := s.GetConfigValue(key)
	if err != nil || val == "" {
		return fallback
	}
	return val
}

func main() {
	log.Printf("Starting HoneyWire Hub v%s initialization...\n", Version)

	cfg := config.Load()

	dbStore, err := store.NewStore(cfg.DBPath)
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize database: %v", err)
	}
	defer dbStore.DB.Close()

	// 1. Establish Root Context for all background workers
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	wsService := websocket.NewService()
	authService := auth.NewService(dbStore, cfg.DashboardPassword)
	nodeSvc := node.NewService(dbStore, wsService)
	sensorSvc := sensor.NewService(dbStore, wsService)
	siemService := siem.NewService()
	notifyService := notify.NewService()
	eventSvc := event.NewService(dbStore, wsService, siemService, notifyService, cfg.Version)
	configService := config.NewService(dbStore, authService, siemService, notifyService, cfg.DashboardPassword, cfg.Version)

	authHandler := api.NewAuthHandler(authService, cfg)
	nodeHandler := api.NewNodeHandler(nodeSvc, authHandler)
	sensorHandler := api.NewSensorHandler(sensorSvc, authHandler, authService)
	eventsHandler := api.NewEventHandler(eventSvc, cfg, authHandler)
	analyticsHandler := api.NewAnalyticsHandler(dbStore)
	configHandler := api.NewConfigHandler(configService, cfg)
	composeHandler := api.NewComposeHandler(dbStore)

	r, err := api.SetupRouter(api.RouterConfig{
		Nodes:        nodeHandler,
		Sensors:      sensorHandler,
		Auth:         authHandler,
		Events:       eventsHandler,
		Analytics:    analyticsHandler,
		Config:       configHandler,
		Compose:      composeHandler,
		WSService:    wsService,
		SessionValidator: authService,
	})
	if err != nil {
		log.Fatalf("[FATAL] Router setup failed: %v", err)
	}

	// 1. Start External Workers
	go api.StartHealthMonitor(rootCtx, dbStore, wsService)
	go wsService.StartChartSyncBroadcaster(rootCtx)
	go authService.StartWorkers(rootCtx)
	go siemService.StartWorker(rootCtx)
	go notifyService.StartWorker(rootCtx)

	// 2. Load Runtime Configurations Safely
	isArmed := loadConfigSafe(dbStore, "is_armed", "false") == "true"
	webhookType := loadConfigSafe(dbStore, "webhook_type", "ntfy")
	webhookURL := loadConfigSafe(dbStore, "webhook_url", "")
	webhookEvents := loadConfigSafe(dbStore, "webhook_events", "[]")
	notifyService.UpdateConfig(isArmed, webhookType, webhookURL, webhookEvents)

	siemAddress := loadConfigSafe(dbStore, "siem_address", "")
	siemProtocol := loadConfigSafe(dbStore, "siem_protocol", "tcp")
	siemService.UpdateConfig(siemAddress, siemProtocol)

	if siemAddress != "" {
		log.Printf("[SIEM] Configured to forward to %s via %s\n", siemAddress, siemProtocol)
	} else {
		log.Println("[SIEM] Forwarding disabled (no address configured).")
	}

	// 3. Start Database Retention Worker (cancelable)
	go startRetentionWorker(rootCtx, dbStore)

	// 4. Start HTTP Server
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("[*] Server listening on port %s\n", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] Server failed to start: %v", err)
		}
	}()

	// 5. Graceful Shutdown Handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	<-sigChan
	log.Println("\n[*] Received shutdown signal. Initiating graceful shutdown...")

	// Signal background workers to stop
	rootCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("[!] Server forced to shut down: %v\n", err)
	}

	log.Println("[*] Flushing pending webhooks...")
	if err := notifyService.FlushQueue(5 * time.Second); err != nil {
		log.Printf("[!] Webhook flush error: %v\n", err)
	}

	log.Println("[*] Flushing pending SIEM events...")
	if err := siemService.FlushQueue(5 * time.Second); err != nil {
		log.Printf("[!] SIEM flush error: %v\n", err)
	}

	log.Println("[*] Graceful shutdown complete. Goodbye!")
}

// startRetentionWorker wakes up periodically to archive/purge old events based on DB settings
func startRetentionWorker(ctx context.Context, s *store.SQLiteStore) {
	// Wake up every hour to check retention
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[Retention] worker stopped")
			return
		case <-ticker.C:
			archiveDays, _ := strconv.Atoi(loadConfigSafe(s, "auto_archive_days", "0"))
			purgeDays, _ := strconv.Atoi(loadConfigSafe(s, "auto_purge_days", "0"))

			if archiveDays > 0 || purgeDays > 0 {
				if err := s.EnforceRetention(archiveDays, purgeDays); err != nil {
					log.Printf("[WARNING] Event retention task failed: %v", err)
				}
			}
		}
	}
}
