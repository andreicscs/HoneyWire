package api

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

func SetupRouter(h *Handler) (*chi.Mux, error) {
	r := chi.NewRouter()

	r.Use(ErrorOnlyLogger)
	r.Use(middleware.Recoverer)

	// Public Endpoints
	r.Get("/api/v1/version", h.HandleVersion)
	r.Post("/login", h.Login)
	r.Post("/logout", h.Logout)
	r.Get("/api/v1/setup/status", h.GetSetupStatus)
	r.Post("/api/v1/setup", h.CompleteSetup)

	// UI Endpoints (Protected by Cookies)
	r.Group(func(r chi.Router) {
		r.Use(UIAuthMiddleware(h.SessionStore))

		r.Get("/api/v1/ws", h.WSService.HandleWS)

		// UI Compose Preview
		r.Post("/api/v1/compose/generate", h.GenerateCompose) // Used by UI modal for live preview

		// --- Node & Fleet Management ---
		r.Post("/api/v1/nodes", h.CreateNode)
		r.Get("/api/v1/nodes", h.GetNodes)
		r.Get("/api/v1/nodes/{nodeId}", h.GetNodeDetails)
		r.Patch("/api/v1/nodes/{nodeId}", h.UpdateNode)
		r.Delete("/api/v1/nodes/{nodeId}", h.DeleteNode)

		// --- Sensor Management ---
		r.Post("/api/v1/nodes/{nodeId}/sensors", h.AddNodeSensor)
		r.Put("/api/v1/nodes/{nodeId}/sensors/{sensorId}", h.EditNodeSensor)
		r.Delete("/api/v1/nodes/{nodeId}/sensors/{sensorId}", h.DeleteNodeSensor)
		r.Patch("/api/v1/nodes/{nodeId}/sensors/{sensorId}/silence", h.ToggleSilence)

		// System Configuration & Danger Zone
		r.Get("/api/v1/config", h.GetConfig)
		r.Patch("/api/v1/config", h.UpdateConfig)
		r.Patch("/api/v1/system/password", h.ChangePassword)
		r.Post("/api/v1/system/reset", h.FactoryReset)

		// Telemetry & State (For UI Dashboards)
		r.Get("/api/v1/events/severity", h.GetSeverityAnalytics)
		r.Get("/api/v1/events/velocity", h.GetVelocityAnalytics)
		r.Get("/api/v1/events", h.GetEvents)
		r.Get("/api/v1/uptime", h.GetUptime)
		r.Get("/api/v1/system/state", h.GetSystemState)
		r.Patch("/api/v1/system/state", h.SetSystemState)

		// Event Management
		r.Get("/api/v1/events/unread", h.GetUnreadCount)
		r.Patch("/api/v1/events/read", h.MarkEventsRead)
		r.Patch("/api/v1/events/{eventId}/read", h.MarkSingleEventRead)
		r.Delete("/api/v1/events", h.ClearEvents)
		r.Patch("/api/v1/events/{eventId}/archive", h.ArchiveEvent)
		r.Patch("/api/v1/events/archive-all", h.ArchiveAll)
	})

	// --- Wizard & Telemetry Endpoints ---
	// Authentication is handled via API Key (Bearer Token) inside the handler
	r.Get("/api/v1/nodes/me", h.GetCurrentNode)      // node whoami based on api key.
	r.Get("/api/v1/nodes/compose", h.GetNodeCompose) // aggreagates all generated compose files for a node's sensors
	r.Post("/api/v1/heartbeat", h.ReceiveHeartbeat)
	r.Post("/api/v1/offline", h.ReceiveOffline)
	r.Post("/api/v1/event", h.ReceiveEvent)

	r.Get("/api/v1/manifests", h.GetManifests) // Fetches catalog (Dual Auth)

	// --- Serve the Vue Frontend ---
	distFS, err := fs.Sub(ui.StaticFiles, "dist")
	if err != nil {
		return nil, fmt.Errorf("failed to mount embedded UI files: %w", err)
	}

	fileServer := http.FileServer(http.FS(distFS))
	r.Handle("/*", fileServer)

	return r, nil
}
