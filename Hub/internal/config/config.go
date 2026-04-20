package config

import (
	"os"
	"strings"
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
		// Default to empty string. If empty, the Vue frontend will force the Setup screen.
		DashboardPassword: getEnv("HW_DASHBOARD_PASSWORD", ""),
		DBPath:            getEnv("HW_DB_PATH", "honeywire.db"),
		Port:              getEnv("HW_PORT", "8080"),
		Version:           getEnv("HW_VERSION", "1.1.0"),
		Env:               getEnv("HW_ENV", "production"),
		TrustProxy:        isProxyTrusted,
	}
}
