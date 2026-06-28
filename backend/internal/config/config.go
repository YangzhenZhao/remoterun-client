package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr              string
	AllowedOrigin     string
	CookieSecure      bool
	DataDir           string
	DatabaseURL       string
	SessionName       string
	SessionSecret     string
	UpstreamTimeout   time.Duration
	BootstrapUsername string
	BootstrapPassword string
}

func Load() (Config, error) {
	cfg := Config{
		Addr:              getEnv("APP_ADDR", ":8080"),
		AllowedOrigin:     getEnv("ALLOWED_ORIGIN", "http://localhost:5173"),
		CookieSecure:      getEnvBool("COOKIE_SECURE", false),
		DataDir:           getEnv("DATA_DIR", "../data"),
		DatabaseURL:       strings.TrimSpace(os.Getenv("DATABASE_URL")),
		SessionName:       getEnv("SESSION_NAME", "remoterun_session"),
		SessionSecret:     strings.TrimSpace(os.Getenv("SESSION_SECRET")),
		UpstreamTimeout:   getEnvDuration("UPSTREAM_TIMEOUT", 60*time.Second),
		BootstrapUsername: strings.TrimSpace(os.Getenv("ADMIN_USERNAME")),
		BootstrapPassword: os.Getenv("ADMIN_PASSWORD"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	if len(cfg.SessionSecret) < 32 {
		return Config{}, fmt.Errorf("SESSION_SECRET must be at least 32 characters")
	}

	return cfg, nil
}

func getEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}
