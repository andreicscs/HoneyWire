package config

import (
	"errors"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/honeywire/hub/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// Config represents immutable infrastructure-level settings.
// Runtime settings (like API keys, webhooks, and SIEM) are managed in SQLite.
type Config struct {
	DashboardPassword string // Optional override. If set, bypasses the Setup UI.
	DBPath            string
	Port              string
	Version           string
	Env               string
	TrustProxy        bool
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func Load() *Config {
	trustProxyStr := strings.ToLower(getEnv("HW_TRUST_PROXY", "false"))
	isProxyTrusted := trustProxyStr == "true" || trustProxyStr == "1"

	return &Config{
		DashboardPassword: getEnv("HW_DASHBOARD_PASSWORD", ""),
		DBPath:            getEnv("HW_DB_PATH", "honeywire.db"),
		Port:              getEnv("HW_PORT", "8080"),
		Version:           getEnv("HW_VERSION", "1.1.0"),
		Env:               getEnv("HW_ENV", "production"),
		TrustProxy:        isProxyTrusted,
	}
}

type Store interface {
	GetConfigValue(key string) (string, error)
	CompleteSetup(adminHash, hubEndpoint string) error
	GetAllConfig() (map[string]string, error)
	UpdateConfigBatch(updates map[string]interface{}) error
	UpdateConfigValue(key, value string) error
	FactoryReset() error
}

type SessionManager interface {
	ClearAllSessions()
}

type SiemService interface {
	UpdateConfig(address, protocol string)
}

type NotifyService interface {
	UpdateConfig(isArmed bool, webhookType, webhookURL, webhookEvents string)
	UpdateIsArmed(isArmed bool)
}

type Service struct {
	store             Store
	sessionManager    SessionManager
	siemService       SiemService
	notifyService     NotifyService
	dashboardPassword string
	version           string
}

func NewService(store Store, sm SessionManager, siemService SiemService, notifyService NotifyService, dashboardPassword, version string) *Service {
	return &Service{
		store:             store,
		sessionManager:    sm,
		siemService:       siemService,
		notifyService:     notifyService,
		dashboardPassword: dashboardPassword,
		version:           version,
	}
}

func (s *Service) GetSetupStatus() (bool, error) {
	if s.dashboardPassword != "" {
		return false, nil
	}
	isSetup, err := s.store.GetConfigValue("is_setup")
	return err != nil || isSetup != "true", nil
}

func (s *Service) CompleteSetup(password, hubEndpoint string) error {
	if s.dashboardPassword != "" {
		return errors.New("setup_locked")
	}

	isSetup, err := s.store.GetConfigValue("is_setup")
	if err == nil && isSetup == "true" {
		return errors.New("already_setup")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.store.CompleteSetup(string(hash), hubEndpoint)
}

func (s *Service) GetConfig() (models.ConfigPayload, error) {
	kv, err := s.store.GetAllConfig()
	if err != nil {
		return models.ConfigPayload{}, err
	}

	archiveDays, _ := strconv.Atoi(kv["auto_archive_days"])
	purgeDays, _ := strconv.Atoi(kv["auto_purge_days"])

	var events []string
	if kv["webhook_events"] != "" {
		events = strings.Split(kv["webhook_events"], ",")
	}

	return models.ConfigPayload{
		HubEndpoint:     kv["hub_endpoint"],
		AutoArchiveDays: archiveDays,
		AutoPurgeDays:   purgeDays,
		WebhookURL:      kv["webhook_url"],
		WebhookType:     kv["webhook_type"],
		WebhookEvents:   events,
		SiemAddress:     kv["siem_address"],
		SiemProtocol:    kv["siem_protocol"],
	}, nil
}

func (s *Service) UpdateConfig(req map[string]interface{}) error {
	if err := s.store.UpdateConfigBatch(req); err != nil {
		return err
	}

	// Hot reload if related settings were changed
	var siemAddress, siemProtocol string
	if val, ok := req["siem_address"].(string); ok {
		siemAddress = val
	} else {
		siemAddress, _ = s.store.GetConfigValue("siem_address")
	}
	if val, ok := req["siem_protocol"].(string); ok {
		siemProtocol = val
	} else {
		siemProtocol, _ = s.store.GetConfigValue("siem_protocol")
	}
	s.siemService.UpdateConfig(siemAddress, siemProtocol)

	isArmed, _ := s.store.GetConfigValue("is_armed")
	webhookType, _ := s.store.GetConfigValue("webhook_type")
	webhookURL, _ := s.store.GetConfigValue("webhook_url")
	webhookEvents, _ := s.store.GetConfigValue("webhook_events")
	s.notifyService.UpdateConfig(isArmed == "true", webhookType, webhookURL, webhookEvents)

	return nil
}

func (s *Service) ChangePassword(currentPassword, newPassword string) error {
	if s.dashboardPassword != "" {
		return errors.New("password_locked")
	}

	dbHash, err := s.store.GetConfigValue("admin_hash")
	if err != nil {
		return errors.New("database_error")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(dbHash), []byte(currentPassword)); err != nil {
		return errors.New("incorrect_password")
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("hash_error")
	}

	if err := s.store.UpdateConfigValue("admin_hash", string(newHash)); err != nil {
		return errors.New("database_error")
	}

	s.sessionManager.ClearAllSessions()
	return nil
}

func (s *Service) FactoryReset(password, ip string) error {
	dbHash, err := s.store.GetConfigValue("admin_hash")
	if err != nil {
		return errors.New("database_error")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(dbHash), []byte(password)); err != nil {
		return errors.New("incorrect_password")
	}

	log.Printf("[!] AUDIT: IP %s initiated a full Factory Reset. Wiping database.", ip)

	if err := s.store.FactoryReset(); err != nil {
		return errors.New("database_error")
	}

	s.sessionManager.ClearAllSessions()
	return nil
}

func (s *Service) GetSystemState() (bool, error) {
	isArmedStr, err := s.store.GetConfigValue("is_armed")
	if err != nil {
		return true, nil // Default fallback
	}
	return isArmedStr == "true", nil
}

func (s *Service) SetSystemState(isArmed bool) error {
	val := "false"
	if isArmed {
		val = "true"
	}

	if err := s.store.UpdateConfigValue("is_armed", val); err != nil {
		return err
	}

	s.notifyService.UpdateIsArmed(isArmed)
	return nil
}

func (s *Service) GetVersion() string {
	return s.version
}
