package config

import "os"

type AppConfig struct {
	RSSHubURL   string
	DatabaseURL string
	Port        string
	ConfigPath  string
	Loglevel    string
}

func getEnvDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func LoadAppConfig() (*AppConfig, error) {
	cfg := &AppConfig{
		RSSHubURL:   os.Getenv("RSSHUB_BASE_URL"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        getEnvDefault("APP_PORT", "8080"),
		ConfigPath:  getEnvDefault("CONFIG_PATH", "../../configs"),
		Loglevel:    getEnvDefault("LOG_LEVEL", "info"),
	}
	return cfg, nil
}
