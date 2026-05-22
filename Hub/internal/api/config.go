package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/honeywire/hub/internal/auth"
	"github.com/honeywire/hub/internal/models"
	"github.com/honeywire/hub/internal/notify"
	"github.com/honeywire/hub/internal/siem"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) GetSetupStatus(w http.ResponseWriter, r *http.Request) {
	if h.Cfg.DashboardPassword != "" {
		SendJSON(w, http.StatusOK, map[string]bool{"requires_setup": false})
		return
	}
	isSetup, err := h.Store.GetConfigValue("is_setup")
	SendJSON(w, http.StatusOK, map[string]bool{"requires_setup": err != nil || isSetup != "true"})
}

func (h *Handler) CompleteSetup(w http.ResponseWriter, r *http.Request) {
	if h.Cfg.DashboardPassword != "" {
		RespondError(w, "Setup is locked by environment configuration.", http.StatusForbidden)
		return
	}

	isSetup, err := h.Store.GetConfigValue("is_setup")
	if err == nil && isSetup == "true" {
		RespondError(w, "Setup has already been completed. Unauthorized.", http.StatusForbidden)
		return
	}

	var req models.SetupPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if req.HubEndpoint == "" || req.Password == "" {
		RespondError(w, "Invalid setup parameters. Missing required fields.", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		RespondError(w, "Failed to secure password", http.StatusInternalServerError)
		return
	}

	if err := h.Store.CompleteSetup(string(hash), req.HubEndpoint); err != nil {
		RespondError(w, "Failed to complete setup", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	kv, err := h.Store.GetAllConfig()
	if err != nil {
		RespondError(w, "Failed to fetch config", http.StatusInternalServerError)
		return
	}

	archiveDays, _ := strconv.Atoi(kv["auto_archive_days"])
	purgeDays, _ := strconv.Atoi(kv["auto_purge_days"])

	var events []string
	if kv["webhook_events"] != "" {
		events = strings.Split(kv["webhook_events"], ",")
	}

	cfg := models.ConfigPayload{
		HubEndpoint:     kv["hub_endpoint"],
		AutoArchiveDays: archiveDays,
		AutoPurgeDays:   purgeDays,
		WebhookURL:      kv["webhook_url"],
		WebhookType:     kv["webhook_type"],
		WebhookEvents:   events,
		SiemAddress:     kv["siem_address"],
		SiemProtocol:    kv["siem_protocol"],
	}

	SendJSON(w, http.StatusOK, cfg)
}

func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if err := h.Store.UpdateConfigBatch(req); err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Hot reload if related settings were changed
	var siemAddress, siemProtocol string
	if val, ok := req["siem_address"].(string); ok {
		siemAddress = val
	} else {
		siemAddress, _ = h.Store.GetConfigValue("siem_address")
	}
	if val, ok := req["siem_protocol"].(string); ok {
		siemProtocol = val
	} else {
		siemProtocol, _ = h.Store.GetConfigValue("siem_protocol")
	}
	siem.UpdateConfig(siemAddress, siemProtocol)

	isArmed, _ := h.Store.GetConfigValue("is_armed")
	webhookType, _ := h.Store.GetConfigValue("webhook_type")
	webhookURL, _ := h.Store.GetConfigValue("webhook_url")
	webhookEvents, _ := h.Store.GetConfigValue("webhook_events")
	notify.UpdateConfig(isArmed == "true", webhookType, webhookURL, webhookEvents)

	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if h.Cfg.DashboardPassword != "" {
		RespondError(w, "Password is locked by environment configuration.", http.StatusForbidden)
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	dbHash, err := h.Store.GetConfigValue("admin_hash")
	if err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(dbHash), []byte(req.CurrentPassword)); err != nil {
		RespondError(w, "Incorrect current password", http.StatusUnauthorized)
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		RespondError(w, "Failed to hash new password", http.StatusInternalServerError)
		return
	}

	if err := h.Store.UpdateConfigValue("admin_hash", string(newHash)); err != nil {
		RespondError(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	h.SessionStore.ClearAllSessions()
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})

	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *Handler) FactoryReset(w http.ResponseWriter, r *http.Request) {
	// Decode the incoming JSON payload for the password
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Retrieve the master password hash from the database
	dbHash, err := h.Store.GetConfigValue("admin_hash")
	if err != nil {
		RespondError(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Verify the password
	if err := bcrypt.CompareHashAndPassword([]byte(dbHash), []byte(req.Password)); err != nil {
		RespondError(w, "Incorrect password", http.StatusUnauthorized)
		return
	}

	// Proceed with Factory Reset
	ip := h.getRealIP(r)
	log.Printf("[!] AUDIT: IP %s initiated a full Factory Reset. Wiping database.", ip)

	if err := h.Store.FactoryReset(); err != nil {
		RespondError(w, "Failed to factory reset", http.StatusInternalServerError)
		return
	}

	// Terminate all sessions and clear the UI cookie
	h.SessionStore.ClearAllSessions()
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})

	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *Handler) GetSystemState(w http.ResponseWriter, r *http.Request) {
	isArmedStr, err := h.Store.GetConfigValue("is_armed")
	if err != nil {
		isArmedStr = "true" // Default fallback
	}
	SendJSON(w, http.StatusOK, map[string]bool{"is_armed": isArmedStr == "true"})
}

func (h *Handler) SetSystemState(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IsArmed bool `json:"is_armed"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, err.Error(), http.StatusBadRequest)
		return
	}

	val := "false"
	if req.IsArmed {
		val = "true"
	}

	if err := h.Store.UpdateConfigValue("is_armed", val); err != nil {
		RespondError(w, "Failed to update state", http.StatusInternalServerError)
		return
	}

	notify.UpdateIsArmed(req.IsArmed)
	SendJSON(w, http.StatusOK, map[string]interface{}{"status": "success", "is_armed": req.IsArmed})
}

func (h *Handler) HandleVersion(w http.ResponseWriter, r *http.Request) {
	SendJSON(w, http.StatusOK, map[string]string{"version": h.Cfg.Version})
}
