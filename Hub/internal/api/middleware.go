package api

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/honeywire/hub/internal/auth"
	"github.com/honeywire/hub/internal/store"
)

// UIAuthMiddleware strictly requires a valid session cookie for ALL dashboard routes
func UIAuthMiddleware(sessionStore *auth.SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(auth.CookieName)
			if err != nil || !sessionStore.IsValid(cookie.Value) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// AgentAuthMiddleware securely validates sensor heartbeats/events against the DB Config
func AgentAuthMiddleware(s *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var configuredKey string
			err := s.DB.QueryRow("SELECT value FROM config WHERE key='hub_key'").Scan(&configuredKey)
			if err != nil || configuredKey == "" {
				http.Error(w, "Hub is not fully configured.", http.StatusServiceUnavailable)
				return
			}

			token := r.Header.Get("X-Api-Key")
			if token == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					token = strings.TrimSpace(authHeader[7:])
				}
			}

			//Constant Time Compare prevents timing-attack vulnerability scanning
			if token == "" || len(token) != len(configuredKey) || subtle.ConstantTimeCompare([]byte(token), []byte(configuredKey)) != 1 {
				http.Error(w, "Unauthorized Sensor", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}