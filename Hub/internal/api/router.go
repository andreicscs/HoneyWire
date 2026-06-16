package api

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/honeywire/hub/internal/services/websocket"
	"github.com/honeywire/hub/ui"
)

func ErrorOnlyLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()

		next.ServeHTTP(ww, r)

		if ww.Status() >= 400 {
			log.Printf("[-] HTTP %d | %s %s from %s (took %v)",
				ww.Status(), r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
		}
	})
}

type RouterConfig struct {
	Nodes             *NodeHandler
	Sensors           *SensorHandler
	Auth              *AuthHandler
	Events            *EventHandler
	Analytics         *AnalyticsHandler
	Config            *ConfigHandler
	Compose           *ComposeHandler
	WSService         *websocket.Service
	SessionValidator  SessionValidator
	NodeAuthenticator NodeAuthenticator
}

func SetupRouter(cfg RouterConfig) (*chi.Mux, error) {
	r := chi.NewRouter()

	r.Use(ErrorOnlyLogger)
	r.Use(middleware.Recoverer)

	rateLimiter := NewRateLimiter()

	// Public Endpoints
	r.Get("/api/v1/version", cfg.Config.HandleVersion)
	r.Post("/login", cfg.Auth.Login)
	r.Post("/logout", cfg.Auth.Logout)
	r.Get("/api/v1/setup/status", cfg.Config.GetSetupStatus)
	r.Post("/api/v1/setup", cfg.Config.CompleteSetup)

	// UI Endpoints (Protected by Cookies)
	r.Group(func(r chi.Router) {
		r.Use(UIAuthMiddleware(cfg.SessionValidator))

		r.Get("/api/v1/ws", cfg.WSService.HandleWS)

		// UI Compose Preview
		r.Post("/api/v1/compose/generate", cfg.Compose.GenerateCompose) // Used by UI modal for live preview

		// --- Node & Fleet Management ---
		r.Post("/api/v1/nodes", cfg.Nodes.CreateNode)
		r.Get("/api/v1/nodes", cfg.Nodes.GetNodes)
		r.Get("/api/v1/nodes/{nodeId}", cfg.Nodes.GetNodeDetails)
		r.Patch("/api/v1/nodes/{nodeId}", cfg.Nodes.UpdateNode)
		r.Post("/api/v1/nodes/{nodeId}/upgrade", cfg.Nodes.UpgradeNode)
		r.Delete("/api/v1/nodes/{nodeId}", cfg.Nodes.DeleteNode)

		// --- Sensor Management ---
		r.Post("/api/v1/nodes/{nodeId}/sensors", cfg.Nodes.AddNodeSensor)
		r.Put("/api/v1/nodes/{nodeId}/sensors/{sensorId}", cfg.Nodes.EditNodeSensor)
		r.Post("/api/v1/nodes/{nodeId}/sensors/{sensorId}/upgrade", cfg.Nodes.UpgradeNodeSensor)
		r.Delete("/api/v1/nodes/{nodeId}/sensors/{sensorId}", cfg.Nodes.DeleteNodeSensor)
		r.Patch("/api/v1/nodes/{nodeId}/sensors/{sensorId}/silence", cfg.Sensors.ToggleSilence)

		// System Configuration & Danger Zone
		r.Get("/api/v1/config", cfg.Config.GetConfig)
		r.Patch("/api/v1/config", cfg.Config.UpdateConfig)
		r.Patch("/api/v1/system/password", cfg.Config.ChangePassword)
		r.Post("/api/v1/system/reset", cfg.Config.FactoryReset)

		// Telemetry & State (For UI Dashboards)
		r.Get("/api/v1/events/severity", cfg.Analytics.GetSeverityAnalytics)
		r.Get("/api/v1/events/velocity", cfg.Analytics.GetVelocityAnalytics)
		r.Get("/api/v1/events/summary", cfg.Analytics.GetSummaryAnalytics)
		r.Get("/api/v1/events", cfg.Events.GetEvents)
		r.Get("/api/v1/uptime", cfg.Analytics.GetUptime)
		r.Get("/api/v1/system/state", cfg.Config.GetSystemState)
		r.Patch("/api/v1/system/state", cfg.Config.SetSystemState)

		// Event Management
		r.Get("/api/v1/events/unread", cfg.Events.GetUnreadCount)
		r.Patch("/api/v1/events/read", cfg.Events.MarkEventsRead)
		r.Patch("/api/v1/events/{eventId}/read", cfg.Events.MarkSingleEventRead)
		r.Delete("/api/v1/events", cfg.Events.ClearEvents)
		r.Patch("/api/v1/events/{eventId}/archive", cfg.Events.ArchiveEvent)
		r.Patch("/api/v1/events/archive-all", cfg.Events.ArchiveAll)
	})

	// --- Wizard & Telemetry Endpoints ---
	r.Group(func(r chi.Router) {
		r.Use(AgentAuthMiddleware(cfg.NodeAuthenticator, rateLimiter))
		r.Get("/api/v1/nodes/me", cfg.Nodes.GetCurrentNode)
		r.Get("/api/v1/nodes/compose", cfg.Compose.GetNodeCompose)
		r.Post("/api/v1/heartbeat", cfg.Sensors.ReceiveHeartbeat)
		r.Post("/api/v1/offline", cfg.Sensors.ReceiveOffline)
		r.Post("/api/v1/event", cfg.Events.ReceiveEvent)
	})

	r.With(DualAuthMiddleware(cfg.SessionValidator, cfg.NodeAuthenticator, rateLimiter)).Group(func(r chi.Router) {
		r.Get("/api/v1/manifests", cfg.Sensors.GetManifests)
		r.Get("/api/v1/manifests/{sensorId}/versions", cfg.Sensors.GetSpecificManifest)
	})

	// --- Serve the Vue Frontend ---
	distFS, err := fs.Sub(ui.StaticFiles, "dist")
	if err != nil {
		return nil, fmt.Errorf("failed to mount embedded UI files: %w", err)
	}

	fileServer := http.FileServer(http.FS(distFS))
	r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Strict API Protection: Never return HTML for missing API routes
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// 2. Check if the static file exists in the embedded filesystem
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "."
		}
		if _, err := fs.Stat(distFS, path); err != nil {
			// 3. SPA Fallback: Serve index.html for frontend routes
			r.URL.Path = "/"
		}

		fileServer.ServeHTTP(w, r)
	}))

	return r, nil
}
