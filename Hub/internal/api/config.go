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
	"github.com/honeywire/hub/internal/store"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) GetSetupStatus(w http.ResponseWriter, r *http.Request) {
	if h.Cfg.DashboardPassword != "" {
		SendJSON(w, http.StatusOK, map[string]bool{"requires_setup": false})
		return
	}
	var isSetup string
	err := h.Store.DB.QueryRow("SELECT value FROM config WHERE key='is_setup'").Scan(&isSetup)
	SendJSON(w, http.StatusOK, map[string]bool{"requires_setup": err != nil || isSetup != "true"})
}

func (h *Handler) CompleteSetup(w http.ResponseWriter, r *http.Request) {
	if h.Cfg.DashboardPassword != "" {
		http.Error(w, "Setup is locked by environment configuration.", http.StatusForbidden)
		return
	}

	var isSetup string
	err := h.Store.DB.QueryRow("SELECT value FROM config WHERE key='is_setup'").Scan(&isSetup)
	if err == nil && isSetup == "true" {
		http.Error(w, "Setup has already been completed. Unauthorized.", http.StatusForbidden)
		return
	}

	var req models.SetupPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if req.HubEndpoint == "" || req.HubKey == "" || req.Password == "" {
		http.Error(w, "Invalid setup parameters. Missing required fields.", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to secure password", http.StatusInternalServerError)
		return
	}

	tx, _ := h.Store.DB.Begin()
	tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES ('admin_hash', ?)", string(hash))
	tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES ('hub_endpoint', ?)", req.HubEndpoint)
	tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES ('hub_key', ?)", req.HubKey)
	tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES ('is_setup', 'true')")
	tx.Commit()

	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Store.DB.Query("SELECT key, value FROM config")
	if err != nil {
		http.Error(w, "Failed to fetch config", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	kv := make(map[string]string)
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		kv[k] = v
	}

	archiveDays, _ := strconv.Atoi(kv["auto_archive_days"])
	purgeDays, _ := strconv.Atoi(kv["auto_purge_days"])

	var events []string
	if kv["webhook_events"] != "" {
		events = strings.Split(kv["webhook_events"], ",")
	}

	cfg := models.ConfigPayload{
		HubEndpoint:     kv["hub_endpoint"],
		HubKey:          kv["hub_key"],
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
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	tx, err := h.Store.DB.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	validWebhooks := map[string]bool{"ntfy": true, "gotify": true, "discord": true, "slack": true}
	validProtocols := map[string]bool{"tcp": true, "udp": true}

	for key, val := range req {
		switch key {
		case "hub_endpoint", "hub_key", "webhook_url", "siem_address":
			if strVal, ok := val.(string); ok {
				tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)", key, strVal)
			}
		case "webhook_type":
			if strVal, ok := val.(string); ok && validWebhooks[strings.ToLower(strVal)] {
				tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)", key, strings.ToLower(strVal))
			}
		case "siem_protocol":
			if strVal, ok := val.(string); ok && validProtocols[strings.ToLower(strVal)] {
				tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)", key, strings.ToLower(strVal))
			}
		case "auto_archive_days", "auto_purge_days":
			if numVal, ok := val.(float64); ok {
				tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)", key, strconv.Itoa(int(numVal)))
			}
		case "webhook_events":
			if arrVal, ok := val.([]interface{}); ok {
				var events []string
				for _, v := range arrVal {
					if str, ok := v.(string); ok {
						events = append(events, str)
					}
				}
				tx.Exec("INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)", key, strings.Join(events, ","))
			}
		}
	}

	tx.Commit()

	// Hot reload if related settings were changed
	var siemAddress, siemProtocol string
	if val, ok := req["siem_address"].(string); ok {
		siemAddress = val
	} else {
		h.Store.DB.QueryRow("SELECT value FROM config WHERE key='siem_address'").Scan(&siemAddress)
	}
	if val, ok := req["siem_protocol"].(string); ok {
		siemProtocol = val
	} else {
		h.Store.DB.QueryRow("SELECT value FROM config WHERE key='siem_protocol'").Scan(&siemProtocol)
	}
	siem.UpdateConfig(siemAddress, siemProtocol)

	var isArmed, webhookType, webhookURL, webhookEvents string
	h.Store.DB.QueryRow("SELECT value FROM config WHERE key='is_armed'").Scan(&isArmed)
	h.Store.DB.QueryRow("SELECT value FROM config WHERE key='webhook_type'").Scan(&webhookType)
	h.Store.DB.QueryRow("SELECT value FROM config WHERE key='webhook_url'").Scan(&webhookURL)
	h.Store.DB.QueryRow("SELECT value FROM config WHERE key='webhook_events'").Scan(&webhookEvents)
	notify.UpdateConfig(isArmed == "true", webhookType, webhookURL, webhookEvents)

	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if h.Cfg.DashboardPassword != "" {
		http.Error(w, "Password is locked by environment configuration.", http.StatusForbidden)
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	var dbHash string
	err := h.Store.DB.QueryRow("SELECT value FROM config WHERE key='admin_hash'").Scan(&dbHash)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(dbHash), []byte(req.CurrentPassword)); err != nil {
		http.Error(w, "Incorrect current password", http.StatusUnauthorized)
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash new password", http.StatusInternalServerError)
		return
	}

	h.Store.DB.Exec("UPDATE config SET value = ? WHERE key = 'admin_hash'", string(newHash))

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
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Retrieve the master password hash from the database
	var dbHash string
	err := h.Store.DB.QueryRow("SELECT value FROM config WHERE key='admin_hash'").Scan(&dbHash)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Verify the password
	if err := bcrypt.CompareHashAndPassword([]byte(dbHash), []byte(req.Password)); err != nil {
		http.Error(w, "Incorrect password", http.StatusUnauthorized)
		return
	}

	// Proceed with Factory Reset
	ip := h.getRealIP(r)
	log.Printf("[!] AUDIT: IP %s initiated a full Factory Reset. Wiping database.", ip)

	tx, _ := h.Store.DB.Begin()
	tx.Exec("DELETE FROM events")
	tx.Exec("DELETE FROM sensors")
	tx.Exec("DELETE FROM sensor_heartbeats")
	tx.Exec("DELETE FROM config")
	tx.Commit()

	store.InitializeDefaultConfig(h.Store.DB)

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
	var isArmedStr string
	h.Store.DB.QueryRow("SELECT value FROM config WHERE key='is_armed'").Scan(&isArmedStr)
	SendJSON(w, http.StatusOK, map[string]bool{"is_armed": isArmedStr == "true"})
}

func (h *Handler) SetSystemState(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IsArmed bool `json:"is_armed"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	val := "false"
	if req.IsArmed {
		val = "true"
	}
	h.Store.DB.Exec("UPDATE config SET value=? WHERE key='is_armed'", val)
	notify.UpdateIsArmed(req.IsArmed)
	SendJSON(w, http.StatusOK, map[string]interface{}{"status": "success", "is_armed": req.IsArmed})
}

func (h *Handler) HandleVersion(w http.ResponseWriter, r *http.Request) {
	SendJSON(w, http.StatusOK, map[string]string{"version": h.Cfg.Version})
}
