package config

import (
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Setenv("CODEX_USAGE_ADDR", ":9090")
	t.Setenv("CODEX_USAGE_ENV", "prod")
	t.Setenv("CODEX_USAGE_DATABASE_URL", "postgres://example")
	t.Setenv("CODEX_USAGE_RAW_DIR", "/tmp/raw")
	t.Setenv("CODEX_USAGE_ADMIN_PASSWORD", "password")
	t.Setenv("CODEX_USAGE_SESSION_SECRET", "session-secret")
	t.Setenv("CODEX_USAGE_DEVICE_TOKENS", " token-a,token-b ")
	t.Setenv("CODEX_USAGE_COOKIE_SECURE", "false")
	t.Setenv("CODEX_USAGE_ALLOWED_ORIGINS", " http://localhost:5173,https://example.com ")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if got.Environment != "prod" || got.Addr != ":9090" || got.DatabaseURL != "postgres://example" || got.RawDir != "/tmp/raw" || got.AdminPassword != "password" {
		t.Fatalf("unexpected config basics: %+v", got)
	}
	if !got.IsProduction() {
		t.Fatalf("expected prod environment: %+v", got)
	}
	if string(got.SessionSecret) != "session-secret" || got.SessionTTL != 7*24*time.Hour || got.CookieSecure {
		t.Fatalf("unexpected session config: %+v", got)
	}
	if _, ok := got.DeviceTokens["token-a"]; !ok {
		t.Fatalf("missing token-a: %+v", got.DeviceTokens)
	}
	if _, ok := got.DeviceTokens["token-b"]; !ok {
		t.Fatalf("missing token-b: %+v", got.DeviceTokens)
	}
	if len(got.AllowedOrigins) != 2 || got.AllowedOrigins[0] != "http://localhost:5173" || got.AllowedOrigins[1] != "https://example.com" {
		t.Fatalf("unexpected origins: %+v", got.AllowedOrigins)
	}
}

func TestConfig_IsProduction(t *testing.T) {
	cases := []struct {
		name        string
		environment string
		want        bool
	}{
		{name: "prod", environment: "prod", want: true},
		{name: "production", environment: "production", want: true},
		{name: "case insensitive", environment: "PROD", want: true},
		{name: "development", environment: "development", want: false},
		{name: "empty", environment: "", want: false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := Config{Environment: tt.environment}.IsProduction()
			if got != tt.want {
				t.Fatalf("IsProduction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoad_RequiresDatabasePasswordAndDeviceTokens(t *testing.T) {
	t.Setenv("CODEX_USAGE_DATABASE_URL", "")
	t.Setenv("CODEX_USAGE_ADMIN_PASSWORD", "password")
	t.Setenv("CODEX_USAGE_DEVICE_TOKENS", "token")
	if _, err := Load(); err == nil {
		t.Fatalf("expected database url error")
	}

	t.Setenv("CODEX_USAGE_DATABASE_URL", "postgres://example")
	t.Setenv("CODEX_USAGE_ADMIN_PASSWORD", "")
	t.Setenv("CODEX_USAGE_DEVICE_TOKENS", "token")
	if _, err := Load(); err == nil {
		t.Fatalf("expected admin password error")
	}

	t.Setenv("CODEX_USAGE_DATABASE_URL", "postgres://example")
	t.Setenv("CODEX_USAGE_ADMIN_PASSWORD", "password")
	t.Setenv("CODEX_USAGE_DEVICE_TOKENS", "")
	if _, err := Load(); err == nil {
		t.Fatalf("expected device token error")
	}
}
