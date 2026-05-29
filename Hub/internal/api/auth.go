package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/honeywire/hub/internal/services/auth"
	"github.com/honeywire/hub/internal/services/config"
)

const AuthCookieName = "hw_auth"

type AuthHandler struct {
	service *auth.Service
	Cfg     *config.Config
}

func NewAuthHandler(svc *auth.Service, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		service: svc,
		Cfg:     cfg,
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ip := GetRealIP(r, h.Cfg.TrustProxy)

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	token, err := h.service.Login(req.Password, ip)
	if err != nil {
		if err.Error() == "too_many_requests" {
			RespondError(w, "Too many failed attempts. Try again later.", http.StatusTooManyRequests)
			return
		}
		RespondError(w, "Invalid Password", http.StatusUnauthorized)
		return
	}

	isProd := h.Cfg.Env == "production"
	http.SetCookie(w, &http.Cookie{
		Name:     AuthCookieName,
		Value:    token,
		MaxAge:   2592000,
		HttpOnly: true,
		Secure:   isProd,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	SendJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(AuthCookieName); err == nil {
		h.service.DeleteSession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     AuthCookieName,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		Path:     "/",
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// AuthenticateNodeRequest is used by other handlers to validate agent requests.
func (h *AuthHandler) AuthenticateNodeRequest(r *http.Request) (string, error) {

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid authorization format")
	}

	return h.service.AuthenticateNodeRequest(parts[1])
}
