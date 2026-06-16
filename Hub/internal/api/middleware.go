package api

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/mod/semver"

	"github.com/honeywire/hub/internal/models"
	"golang.org/x/time/rate"
)

type SessionValidator interface {
	IsValid(string) bool
}

type NodeAuthenticator interface {
	AuthenticateNodeRequest(token string) (string, error)
}

type contextKey string

const (
	NodeIDKey     contextKey = "nodeID"
)

// RateLimiter controls the frequency of requests on a per-node basis.
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
	}
}

// Allow checks if a request from a given nodeID is permitted.
func (rl *RateLimiter) Allow(nodeID string) bool {
	rl.mu.RLock()
	limiter, exists := rl.limiters[nodeID]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		limiter, exists = rl.limiters[nodeID]
		if !exists {
			// Limit: ~1.66 requests per second (100 per minute), Burst: 100
			limiter = rate.NewLimiter(rate.Limit(100.0/60.0), 100)
			rl.limiters[nodeID] = limiter
		}
		rl.mu.Unlock()
	}

	return limiter.Allow()
}

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
func AgentAuthMiddleware(auth NodeAuthenticator, rateLimiter *RateLimiter) func(http.Handler) http.Handler {
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

			if !rateLimiter.Allow(nodeID) {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			// Version handshake
			wizardMinHubAPIStr := r.Header.Get("X-Wizard-Min-Hub-Api")
			if wizardMinHubAPIStr != "" {
				reqVer := strings.TrimSpace(wizardMinHubAPIStr)
				if !strings.HasPrefix(reqVer, "v") { reqVer = "v" + reqVer }
				curVer := models.HubVersion
				if !strings.HasPrefix(curVer, "v") { curVer = "v" + curVer }

				if !semver.IsValid(reqVer) {
					http.Error(w, "Invalid X-Wizard-Min-Hub-Api format", http.StatusBadRequest)
					return
				}
				if semver.Compare(curVer, reqVer) < 0 {
					http.Error(w, "This Wizard requires Hub "+wizardMinHubAPIStr+" or later. Please update your Hub.", http.StatusUpgradeRequired)
					return
				}
			}

			ctx := context.WithValue(r.Context(), NodeIDKey, nodeID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// DualAuthMiddleware permits either UI Dashboard Sessions OR Agent API Keys
func DualAuthMiddleware(sessionValidator SessionValidator, auth NodeAuthenticator, rateLimiter *RateLimiter) func(http.Handler) http.Handler {
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
					if !rateLimiter.Allow(nodeID) {
						http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
						return
					}

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
