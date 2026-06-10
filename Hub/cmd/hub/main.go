package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"sync"

	"github.com/honeywire/hub/internal/api"
	"github.com/honeywire/hub/internal/services/auth"
	composesvc "github.com/honeywire/hub/internal/services/compose"
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
	siemService := siem.NewService(nodeSvc)
	notifyService := notify.NewService()
	eventSvc := event.NewService(dbStore, wsService, siemService, notifyService, cfg.Version)
	configService := config.NewService(dbStore, authService, siemService, notifyService, cfg.DashboardPassword, cfg.Version)
	composeService := composesvc.NewService(dbStore)

	authHandler := api.NewAuthHandler(authService, cfg)
	nodeHandler := api.NewNodeHandler(nodeSvc)
	sensorHandler := api.NewSensorHandler(sensorSvc, composeService)
	eventsHandler := api.NewEventHandler(eventSvc, cfg)
	analyticsHandler := api.NewAnalyticsHandler(dbStore)
	configHandler := api.NewConfigHandler(configService, cfg)
	composeHandler := api.NewComposeHandler(composeService)

	r, err := api.SetupRouter(api.RouterConfig{
		Nodes:             nodeHandler,
		Sensors:           sensorHandler,
		Auth:              authHandler,
		Events:            eventsHandler,
		Analytics:         analyticsHandler,
		Config:            configHandler,
		Compose:           composeHandler,
		WSService:         wsService,
		SessionValidator:  authService,
		NodeAuthenticator: authService,
	})
	if err != nil {
		log.Fatalf("[FATAL] Router setup failed: %v", err)
	}

	// 1. Start External Workers
	go sensorSvc.StartHealthMonitor(rootCtx)
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

	go eventSvc.StartRetentionWorker(rootCtx)

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

	// 1. Signal all background workers to stop accepting new work and begin draining
	rootCancel()

	// 2. Shut down the HTTP server (stops taking new HTTP requests)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("[!] Server forced to shut down: %v\n", err)
	}

	// 3. Wait for critical telemetry pipelines to flush
	// We wait for them concurrently to speed up the shutdown process.
	// Since both services have hard internal 5-second drain timeouts, 
	// this is guaranteed to never hang the system indefinitely.
	log.Println("[*] Waiting for SIEM and Notification workers to flush remaining events...")
	
	var shutdownWg sync.WaitGroup
	
	shutdownWg.Add(1)
	go func() {
		defer shutdownWg.Done()
		siemService.Wait()
	}()
	
	shutdownWg.Add(1)
	go func() {
		defer shutdownWg.Done()
		notifyService.Wait()
	}()

	// Block until both services confirm their WaitGroups are zero
	shutdownWg.Wait()

	log.Println("[*] Graceful shutdown complete. Goodbye!")
}
