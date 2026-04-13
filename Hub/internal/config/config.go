package config

import (
	"os"
	"strings"
)

type Config struct {
	APISecret         string
	DashboardPassword string
	DBPath            string
	NtfyURL           string
	GotifyURL         string
	GotifyToken       string
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
	// Parse the TrustProxy boolean safely
	trustProxyStr := strings.ToLower(getEnv("HW_TRUST_PROXY", "false"))
	isProxyTrusted := trustProxyStr == "true" || trustProxyStr == "1"

	return &Config{
		APISecret:         getEnv("HW_HUB_KEY", "change_this_to_a_secure_random_string"),
		DashboardPassword: getEnv("HW_DASHBOARD_PASSWORD", "admin"),
		DBPath:            getEnv("HW_DB_PATH", "test_honeywire.db"), // Defaulting to local for testing
		NtfyURL:           getEnv("HW_NTFY_URL", ""),
		GotifyURL:         getEnv("HW_GOTIFY_URL", ""),
		GotifyToken:       getEnv("HW_GOTIFY_TOKEN", ""),
		Port:              getEnv("HW_PORT", "8080"),
		Version:           getEnv("HW_VERSION", "1.0.0"),
		Env:               getEnv("HW_ENV", "development"), // Defaults to development (set the Secure flag on the hw_auth cookie in production)
		TrustProxy:        isProxyTrusted,                  // Defaults to false (trust the X-Forwarded-For header only if explicitly configured to do so)
	}
}