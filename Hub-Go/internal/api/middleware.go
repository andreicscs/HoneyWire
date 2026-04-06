package api

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/honeywire/hub/internal/auth"
	"github.com/honeywire/hub/internal/config"
)

// UIAuthMiddleware requires a valid session cookie
func UIAuthMiddleware(cfg *config.Config, store *auth.SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.DashboardPassword != "" {
				cookie, err := r.Cookie(auth.CookieName)
				if err != nil || !store.IsValid(cookie.Value) {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			}
			// If authorized, pass the request to the actual endpoint
			next.ServeHTTP(w, r)
		})
	}
}

// AgentAuthMiddleware requires the correct API Secret from the sensors
func AgentAuthMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("X-Api-Key")
			
			// Fallback to Bearer token
			if token == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					token = strings.TrimSpace(authHeader[7:])
				}
			}

			// subtle.ConstantTimeCompare prevents timing attacks
			if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(cfg.APISecret)) != 1 {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}