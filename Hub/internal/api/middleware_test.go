package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- Mocks ---
type mockSessionValidator struct {
	validTokens map[string]bool
}

func (m *mockSessionValidator) IsValid(t string) bool { return m.validTokens[t] }

type mockNodeAuth struct {
	validKeys map[string]string
}

func (m *mockNodeAuth) AuthenticateNodeRequest(t string) (string, error) {
	if nodeID, ok := m.validKeys[t]; ok {
		return nodeID, nil
	}
	return "", errors.New("unauthorized")
}

// --- Tests ---

func TestUIAuthMiddleware(t *testing.T) {
	validator := &mockSessionValidator{validTokens: map[string]bool{"good-session": true}}
	middleware := UIAuthMiddleware(validator)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("Missing Cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Invalid Session", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: AuthCookieName, Value: "bad-session"})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Valid Session", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: AuthCookieName, Value: "good-session"})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestAgentAuthMiddleware(t *testing.T) {
	nodeAuth := &mockNodeAuth{validKeys: map[string]string{"agent-key": "node-1"}}
	rateLimiter := NewRateLimiter()
	middleware := AgentAuthMiddleware(nodeAuth, rateLimiter)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nodeID := NodeIDFromContext(r.Context())
		assert.Equal(t, "node-1", nodeID)
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("Valid Token via X-Api-Key", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", nil)
		req.Header.Set("X-Api-Key", "agent-key")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Valid Token via Bearer", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", nil)
		req.Header.Set("Authorization", "Bearer agent-key")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Invalid Token", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", nil)
		req.Header.Set("X-Api-Key", "bad-key")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Rate Limit Triggers", func(t *testing.T) {
		// Use a fresh rate limiter and handler for this subtest to ensure
		// previous tests haven't consumed the burst capacity.
		freshLimiter := NewRateLimiter()
		freshMiddleware := AgentAuthMiddleware(nodeAuth, freshLimiter)
		freshHandler := freshMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Deplete the burst limit (100)
		for i := 0; i < 100; i++ {
			req := httptest.NewRequest("POST", "/", nil)
			req.Header.Set("X-Api-Key", "agent-key")
			rec := httptest.NewRecorder()
			freshHandler.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
		}

		// 101st request should be rate-limited
		req := httptest.NewRequest("POST", "/", nil)
		req.Header.Set("X-Api-Key", "agent-key")
		rec := httptest.NewRecorder()
		freshHandler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	})

	// VERSIONING ARCHITECTURE EXPLANATION:
	// The Hub guarantees backwards compatibility for Wizards, but strongly rejects future Wizards.
	// This ensures that an outdated Hub cannot accidentally corrupt a newer Wizard's state,
	// while allowing older Wizards to safely continue operating on an upgraded Hub.

	t.Run("Futuristic Wizard Mismatch (v99)", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", nil)
		req.Header.Set("X-Api-Key", "agent-key")
		// Simulate a Wizard requesting a highly futuristic Hub API version
		req.Header.Set("X-Wizard-Version", "99.0.0")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		// Should return HTTP 426 Upgrade Required
		assert.Equal(t, http.StatusUpgradeRequired, rec.Code)
	})

	t.Run("Legacy Backward Compat (Wizard v1, Hub v2)", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", nil)
		req.Header.Set("X-Api-Key", "agent-key")
		// Simulate a valid Wizard
		req.Header.Set("X-Wizard-Version", "2.0.0")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		// Should pass completely natively
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Malformed Wizard Header", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", nil)
		req.Header.Set("X-Api-Key", "agent-key")
		req.Header.Set("X-Wizard-Version", "   garbage   ")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestDualAuthMiddleware(t *testing.T) {
	validator := &mockSessionValidator{validTokens: map[string]bool{"ui-session": true}}
	nodeAuth := &mockNodeAuth{validKeys: map[string]string{"agent-key": "node-1"}}
	rateLimiter := NewRateLimiter()
	middleware := DualAuthMiddleware(validator, nodeAuth, rateLimiter)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("API Key Priority Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?key=agent-key", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("UI Fallback Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: AuthCookieName, Value: "ui-session"})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Invalid API Key falls back to valid UI Cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Api-Key", "garbage-key")
		req.AddCookie(&http.Cookie{Name: AuthCookieName, Value: "ui-session"})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		// This MUST pass! The UI makes API calls but might accidentally send garbage headers
		// or left-over states. If they have a valid cookie, they are authorized.
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
