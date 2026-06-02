package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/honeywire/hub/internal/services/auth"
	"github.com/honeywire/hub/internal/services/config"
	"github.com/stretchr/testify/assert"
)

// We use a mock auth store strictly for the handler injection
type mockStore struct{}

func (m *mockStore) GetConfigValue(k string) (string, error) { return "", nil }
func (m *mockStore) GetNodeByKey(t string) (string, error)   { return "", nil }

func setupAuthTestEnv(env string) (*AuthHandler, *auth.Service) {
	svc := auth.NewService(&mockStore{}, "admin123")
	cfg := &config.Config{Env: env, TrustProxy: false}
	handler := NewAuthHandler(svc, cfg)
	return handler, svc
}

func TestAuthHandler_Login(t *testing.T) {
	handler, _ := setupAuthTestEnv("production")

	t.Run("Valid Login", func(t *testing.T) {
		body := bytes.NewReader([]byte(`{"password": "admin123"}`))
		req := httptest.NewRequest("POST", "/login", body)
		rec := httptest.NewRecorder()

		handler.Login(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"status":"ok"`)

		// Verify Cookie Security Flags
		cookies := rec.Result().Cookies()
		assert.Len(t, cookies, 1)
		cookie := cookies[0]
		assert.Equal(t, AuthCookieName, cookie.Name)
		assert.True(t, cookie.HttpOnly, "Cookie MUST be HttpOnly")
		assert.True(t, cookie.Secure, "Cookie MUST be Secure in production")
		assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite, "Cookie MUST use SameSite=Strict")
	})

	t.Run("Invalid Password", func(t *testing.T) {
		body := bytes.NewReader([]byte(`{"password": "wrong"}`))
		req := httptest.NewRequest("POST", "/login", body)
		rec := httptest.NewRecorder()

		handler.Login(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		body := bytes.NewReader([]byte(`{"password": "adm`)) // Malformed JSON
		req := httptest.NewRequest("POST", "/login", body)
		rec := httptest.NewRecorder()

		handler.Login(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Rate Limited", func(t *testing.T) {
		// Exhaust attempts
		for i := 0; i < 10; i++ {
			body := bytes.NewReader([]byte(`{"password": "wrong"}`))
			req := httptest.NewRequest("POST", "/login", body)
			req.RemoteAddr = "192.168.1.50:12345"
			handler.Login(httptest.NewRecorder(), req)
		}

		// 11th request
		body := bytes.NewReader([]byte(`{"password": "wrong"}`))
		req := httptest.NewRequest("POST", "/login", body)
		req.RemoteAddr = "192.168.1.50:12345" // Same IP
		rec := httptest.NewRecorder()

		handler.Login(rec, req)
		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	handler, svc := setupAuthTestEnv("development")

	t.Run("Clears Cookie and Session", func(t *testing.T) {
		// Create a session directly in the service
		token, _ := svc.CreateSession()

		req := httptest.NewRequest("POST", "/logout", nil)
		req.AddCookie(&http.Cookie{Name: AuthCookieName, Value: token})
		rec := httptest.NewRecorder()

		handler.Logout(rec, req)

		assert.Equal(t, http.StatusSeeOther, rec.Code)

		// Verify cookie deletion
		cookies := rec.Result().Cookies()
		assert.Len(t, cookies, 1)
		assert.Equal(t, AuthCookieName, cookies[0].Name)
		assert.Equal(t, -1, cookies[0].MaxAge, "MaxAge MUST be -1 to clear the cookie")
		assert.Equal(t, "", cookies[0].Value, "Cookie value MUST be empty")

		assert.False(t, svc.IsValid(token), "Backend session MUST be deleted")
	})
}
