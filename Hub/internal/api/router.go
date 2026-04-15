package api

import (
	"log"
	"net/http"
	"time"
	"io/fs"
	
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/honeywire/hub/internal/auth"
	"github.com/honeywire/hub/internal/config"
	"github.com/honeywire/hub/internal/store"
	"github.com/honeywire/hub/ui"
)

func ErrorOnlyLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()

		next.ServeHTTP(ww, r)

		// Only print to the terminal if something went wrong
		if ww.Status() >= 400 {
			log.Printf("[-] HTTP %d | %s %s from %s (took %v)",
				ww.Status(), r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
		}
	})
}

func SetupRouter(cfg *config.Config, s *store.Store, sessionStore *auth.SessionStore) *chi.Mux {
	r := chi.NewRouter()
	
	r.Use(ErrorOnlyLogger)
	r.Use(middleware.Recoverer)

	h := NewHandler(s, cfg, sessionStore)

	// Public Endpoints
	r.Get("/api/v1/version", h.HandleVersion)
	r.Post("/login", h.Login)
	r.Post("/logout", h.Logout)
	r.Get("/api/v1/setup/status", h.GetSetupStatus)
	r.Post("/api/v1/setup", h.CompleteSetup)

	// UI Endpoints (Protected by Cookies)
	r.Group(func(r chi.Router) {
		r.Use(UIAuthMiddleware(sessionStore))

		r.Get("/api/v1/ws", h.ServeWS)

		// System Configuration & Danger Zone
		r.Get("/api/v1/config", h.GetConfig)
		r.Patch("/api/v1/config", h.UpdateConfig)
		r.Patch("/api/v1/system/password", h.ChangePassword)
		r.Post("/api/v1/system/reset", h.FactoryReset)
		
		// Telemetry & State
		r.Get("/api/v1/sensors", h.GetSensors)
		r.Get("/api/v1/events", h.GetEvents)
		r.Get("/api/v1/uptime", h.GetUptime)
		r.Get("/api/v1/system/state", h.GetSystemState)
		r.Patch("/api/v1/system/state", h.SetSystemState)

		// Event Management
		r.Get("/api/v1/events/unread", h.GetUnreadCount)
		r.Patch("/api/v1/events/read", h.MarkEventsRead)
		r.Patch("/api/v1/events/{event_id}/read", h.MarkSingleEventRead)
		r.Delete("/api/v1/events", h.ClearEvents)
		r.Patch("/api/v1/events/{event_id}/archive", h.ArchiveEvent)
		r.Patch("/api/v1/events/archive-all", h.ArchiveAll)
		
		// Sensor Management
		r.Patch("/api/v1/sensors/{sensor_id}/silence", h.ToggleSilence)
		r.Delete("/api/v1/sensors/{sensor_id}", h.ForgetSensor)
	})

	// Sensor Endpoints (Protected by API Key)
	r.Group(func(r chi.Router) {
		r.Use(AgentAuthMiddleware(s))
		r.Post("/api/v1/heartbeat", h.ReceiveHeartbeat)
		r.Post("/api/v1/event", h.ReceiveEvent)
	})

	// --- Serve the Vue Frontend ---
	distFS, err := fs.Sub(ui.StaticFiles, "dist")
	if err != nil {
		panic("Failed to mount embedded UI files: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(distFS))
	r.Handle("/*", fileServer)

	return r
}