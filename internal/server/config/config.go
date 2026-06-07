package config

import (
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Environment    string
	Addr           string
	DatabaseURL    string
	RawDir         string
	AdminPassword  string
	SessionSecret  []byte
	SessionTTL     time.Duration
	DeviceTokens   map[string]struct{}
	CookieSecure   bool
	AllowedOrigins []string
	MaxUploadBytes int64
}

func Load() (Config, error) {
	cfg := Config{
		Environment:    envOrDefault("CODEX_USAGE_ENV", "development"),
		Addr:           envOrDefault("CODEX_USAGE_ADDR", ":8080"),
		DatabaseURL:    os.Getenv("CODEX_USAGE_DATABASE_URL"),
		RawDir:         envOrDefault("CODEX_USAGE_RAW_DIR", "/var/lib/codex-usage/raw"),
		AdminPassword:  os.Getenv("CODEX_USAGE_ADMIN_PASSWORD"),
		SessionTTL:     7 * 24 * time.Hour,
		DeviceTokens:   parseCSVSet(os.Getenv("CODEX_USAGE_DEVICE_TOKENS")),
		CookieSecure:   envBool("CODEX_USAGE_COOKIE_SECURE", true),
		AllowedOrigins: parseCSV(os.Getenv("CODEX_USAGE_ALLOWED_ORIGINS")),
		MaxUploadBytes: 512 << 20,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("CODEX_USAGE_DATABASE_URL is required")
	}
	if cfg.AdminPassword == "" {
		return Config{}, fmt.Errorf("CODEX_USAGE_ADMIN_PASSWORD is required")
	}
	if len(cfg.DeviceTokens) == 0 {
		return Config{}, fmt.Errorf("CODEX_USAGE_DEVICE_TOKENS is required")
	}

	secret := os.Getenv("CODEX_USAGE_SESSION_SECRET")
	if secret == "" {
		generated := make([]byte, 32)
		if _, err := rand.Read(generated); err != nil {
			return Config{}, fmt.Errorf("generate session secret: %w", err)
		}
		cfg.SessionSecret = generated
	} else {
		cfg.SessionSecret = []byte(secret)
	}

	return cfg, nil
}

func (c Config) IsProduction() bool {
	environment := strings.TrimSpace(strings.ToLower(c.Environment))
	return environment == "prod" || environment == "production"
}

func envOrDefault(key string, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	return v == "1" || v == "true" || v == "yes"
}

func parseCSVSet(raw string) map[string]struct{} {
	values := parseCSV(raw)
	out := make(map[string]struct{}, len(values))
	for _, v := range values {
		out[v] = struct{}{}
	}
	return out
}

func parseCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
