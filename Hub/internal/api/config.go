package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/honeywire/hub/internal/models"
	"github.com/honeywire/hub/internal/services/config"
)

type ConfigHandler struct {
	service *config.Service
	Cfg     *config.Config
}

func NewConfigHandler(svc *config.Service, cfg *config.Config) *ConfigHandler {
	return &ConfigHandler{service: svc, Cfg: cfg}
}

func (h *ConfigHandler) GetSetupStatus(w http.ResponseWriter, r *http.Request) {
	requiresSetup, _ := h.service.GetSetupStatus()
	SendJSON(w, http.StatusOK, map[string]bool{"requiresSetup": requiresSetup})
}

func (h *ConfigHandler) CompleteSetup(w http.ResponseWriter, r *http.Request) {
	var req models.SetupPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if req.HubEndpoint == "" || req.Password == "" {
		RespondError(w, "Invalid setup parameters. Missing required fields.", http.StatusBadRequest)
		return
	}

	if err := h.service.CompleteSetup(req.Password, req.HubEndpoint); err != nil {
		if err.Error() == "setup_locked" {
			RespondError(w, "Setup is locked by environment configuration.", http.StatusForbidden)
			return
		}
		if err.Error() == "already_setup" {
			RespondError(w, "Setup has already been completed. Unauthorized.", http.StatusForbidden)
			return
		}
		RespondError(w, "Failed to complete setup", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.service.GetConfig()
	if err != nil {
		RespondError(w, "Failed to fetch config", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, map[string]interface{}{
		"hubEndpoint":     cfg.HubEndpoint,
		"registryUrl":     cfg.RegistryURL,
		"autoArchiveDays": cfg.AutoArchiveDays,
		"autoPurgeDays":   cfg.AutoPurgeDays,
		"webhookType":     cfg.WebhookType,
		"webhookUrl":      cfg.WebhookURL,
		"webhookEvents":   cfg.WebhookEvents,
		"siemAddress":     cfg.SiemAddress,
		"siemProtocol":    cfg.SiemProtocol,
		"whitelistedSources": cfg.WhitelistedSources,
	})
}

func (h *ConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateConfig(req); err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *ConfigHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if err := h.service.ChangePassword(req.CurrentPassword, req.NewPassword); err != nil {
		if err.Error() == "password_locked" {
			RespondError(w, "Password is locked by environment configuration.", http.StatusForbidden)
			return
		}
		if err.Error() == "incorrect_password" {
			RespondError(w, "Incorrect current password", http.StatusUnauthorized)
			return
		}
		RespondError(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	// codeql[go/insecure-cookie] Clearing configuration cookie.
	// nosemgrep: go.lang.security.audit.net.cookie-missing-secure.cookie-missing-secure
	http.SetCookie(w, &http.Cookie{
		Name:     AuthCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})

	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *ConfigHandler) FactoryReset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	ip := GetRealIP(r, h.Cfg.TrustProxy)
	dryrun := r.URL.Query().Get("dryrun") == "true"

	if dryrun {
		stats, err := h.service.FactoryResetDryRun(req.Password)
		if err != nil {
			if err.Error() == "incorrect_password" {
				RespondError(w, "Incorrect password", http.StatusUnauthorized)
				return
			}
			RespondError(w, "Failed to calculate dry run", http.StatusInternalServerError)
			return
		}
		SendJSON(w, http.StatusOK, map[string]interface{}{
			"status": "success",
			"dryrun": true,
			"stats":  stats,
		})
		return
	}

	if err := h.service.FactoryReset(req.Password, ip); err != nil {
		if err.Error() == "incorrect_password" {
			RespondError(w, "Incorrect password", http.StatusUnauthorized)
			return
		}
		RespondError(w, "Failed to factory reset", http.StatusInternalServerError)
		return
	}

	// codeql[go/insecure-cookie] Clearing configuration cookie.
	// nosemgrep: go.lang.security.audit.net.cookie-missing-secure.cookie-missing-secure
	http.SetCookie(w, &http.Cookie{
		Name:     AuthCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})

	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *ConfigHandler) GetSystemState(w http.ResponseWriter, r *http.Request) {
	isArmed, _ := h.service.GetSystemState()
	SendJSON(w, http.StatusOK, map[string]bool{"isArmed": isArmed})
}

func (h *ConfigHandler) SetSystemState(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IsArmed bool `json:"isArmed"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.SetSystemState(req.IsArmed); err != nil {
		RespondError(w, "Failed to update state", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, map[string]interface{}{"status": "success", "isArmed": req.IsArmed})
}

func (h *ConfigHandler) HandleVersion(w http.ResponseWriter, r *http.Request) {
	SendJSON(w, http.StatusOK, map[string]string{"version": h.service.GetVersion()})
}
