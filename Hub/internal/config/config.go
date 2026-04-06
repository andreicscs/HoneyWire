package config

import "os"

type Config struct {
	APISecret         string
	DashboardPassword string
	DBPath            string
	NtfyURL           string
	GotifyURL         string
	GotifyToken       string
	Port              string
	Version           string
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func Load() *Config {
	return &Config{
		APISecret:         getEnv("HW_HUB_KEY", "change_this_to_a_secure_random_string"),
		DashboardPassword: getEnv("HW_DASHBOARD_PASSWORD", "admin"),
		DBPath:            getEnv("HW_DB_PATH", "test_honeywire.db"), // Defaulting to local for testing
		NtfyURL:           getEnv("HW_NTFY_URL", ""),
		GotifyURL:         getEnv("HW_GOTIFY_URL", ""),
		GotifyToken:       getEnv("HW_GOTIFY_TOKEN", ""),
		Port:              getEnv("HW_PORT", "8080"),
		Version:           getEnv("HW_VERSION", "1.0.0"),
	}
}