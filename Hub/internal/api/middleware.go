package api

import (
	"context"
	"net/http"
	"strings"
)

type SessionValidator interface {
	IsValid(string) bool
}

type NodeAuthenticator interface {
	AuthenticateNodeRequest(token string) (string, error)
}

type contextKey string

const NodeIDKey contextKey = "nodeID"

func NodeIDFromContext(ctx context.Context) string {
	val, ok := ctx.Value(NodeIDKey).(string)
	if !ok {
		return ""
	}
	return val
}

// UIAuthMiddleware strictly requires a valid session cookie for ALL dashboard routes
func UIAuthMiddleware(sessionValidator SessionValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(AuthCookieName)
			if err != nil || !sessionValidator.IsValid(cookie.Value) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AgentAuthMiddleware securely validates sensor heartbeats/events via the Auth Service
func AgentAuthMiddleware(auth NodeAuthenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("X-Api-Key")
			if token == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					token = strings.TrimSpace(authHeader[7:])
				}
			}

			nodeID, err := auth.AuthenticateNodeRequest(token)
			if err != nil || nodeID == "" {
				http.Error(w, "Unauthorized Sensor", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), NodeIDKey, nodeID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// DualAuthMiddleware permits either UI Dashboard Sessions OR Agent API Keys
func DualAuthMiddleware(sessionValidator SessionValidator, auth NodeAuthenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// 1. Try Node API Key
			token := r.Header.Get("X-Api-Key")
			if token == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					token = strings.TrimSpace(authHeader[7:])
				} else if r.URL.Query().Get("key") != "" {
					token = r.URL.Query().Get("key")
				}
			}
			if token != "" {
				if nodeID, err := auth.AuthenticateNodeRequest(token); err == nil && nodeID != "" {
					ctx := context.WithValue(r.Context(), NodeIDKey, nodeID)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// 2. Try UI Session Cookie (Fallback)
			cookie, err := r.Cookie(AuthCookieName)
			if err == nil && cookie.Value != "" && sessionValidator.IsValid(cookie.Value) {
				next.ServeHTTP(w, r)
				return
			}

			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		})
	}
}
