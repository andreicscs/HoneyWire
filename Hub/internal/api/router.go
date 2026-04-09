package api

import (
	"net/http"
	"io/fs"
	
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/honeywire/hub/internal/auth"
	"github.com/honeywire/hub/internal/config"
	"github.com/honeywire/hub/internal/store"
	"github.com/honeywire/hub/ui"
)

func SetupRouter(cfg *config.Config, s *store.Store, sessionStore *auth.SessionStore) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	h := NewHandler(s, cfg, sessionStore)

	// Public Endpoints
	r.Get("/api/v1/version", h.HandleVersion)
	r.Post("/login", h.Login)
	r.Get("/logout", h.Logout)

	// UI Endpoints (Protected by Cookies)
	r.Group(func(r chi.Router) {
		r.Use(UIAuthMiddleware(cfg, sessionStore))

		r.Get("/api/v1/sensors", h.GetSensors)
		r.Get("/api/v1/events", h.GetEvents)
		r.Get("/api/v1/uptime", h.GetUptime)

		r.Get("/api/v1/system/state", h.GetSystemState)
		r.Patch("/api/v1/system/state", h.SetSystemState)

		r.Patch("/api/v1/events/read", h.MarkEventsRead)
		r.Patch("/api/v1/events/{event_id}/read", h.MarkSingleEventRead)
		r.Delete("/api/v1/events", h.ClearEvents)

		r.Patch("/api/v1/events/{event_id}/archive", h.ArchiveEvent)
		r.Patch("/api/v1/events/archive-all", h.ArchiveAll)
		r.Patch("/api/v1/sensors/{sensor_id}/silence", h.ToggleSilence)
		r.Delete("/api/v1/sensors/{sensor_id}", h.ForgetSensor)
	})

	// Sensor Endpoints (Protected by API Key)
	r.Group(func(r chi.Router) {
		r.Use(AgentAuthMiddleware(cfg))
		r.Post("/api/v1/heartbeat", h.ReceiveHeartbeat)
		r.Post("/api/v1/event", h.ReceiveEvent)
	})

	// --- Serve the Vue Frontend ---
	
	// Extract the 'dist' sub-folder from the embedded filesystem
	distFS, err := fs.Sub(ui.StaticFiles, "dist")
	if err != nil {
		panic("Failed to mount embedded UI files: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(distFS))

	// Catch-all route: Serve the Vue files for anything not matching the API routes above
	r.Handle("/*", fileServer)

	return r
}