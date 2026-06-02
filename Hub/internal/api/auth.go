package api

import (
	"encoding/json"
	"net/http"

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

	// nosemgrep: go.lang.security.audit.net.cookie-missing-secure.cookie-missing-secure
	// codeql[go/insecure-cookie] Dev environment toggle.
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
