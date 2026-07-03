package config

import (
	"errors"
	"strings"
	"testing"
)

func TestLoad_ReadsEnvVars(t *testing.T) {
	t.Setenv("DB_PATH", "/data/wedding.db")
	t.Setenv("ADMIN_PASSWORD", "secret-pass")
	t.Setenv("SESSION_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("RESEND_API_KEY", "re_test_key")
	t.Setenv("RESEND_FROM", "rsvp@carlaochfrodi.wedding")
	t.Setenv("RESEND_TO", "frodi.carla@gmail.com")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://carlaochfrodi.wedding,https://staging.carlaochfrodi.wedding")
	t.Setenv("PORT", "9090")
	t.Setenv("SECURE_COOKIE", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.DBPath != "/data/wedding.db" {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, "/data/wedding.db")
	}
	if cfg.AdminPassword != "secret-pass" {
		t.Errorf("AdminPassword = %q", cfg.AdminPassword)
	}
	if cfg.SessionSecret != "0123456789abcdef0123456789abcdef" {
		t.Errorf("SessionSecret = %q", cfg.SessionSecret)
	}
	if cfg.ResendAPIKey != "re_test_key" {
		t.Errorf("ResendAPIKey = %q", cfg.ResendAPIKey)
	}
	if cfg.ResendFrom != "rsvp@carlaochfrodi.wedding" {
		t.Errorf("ResendFrom = %q", cfg.ResendFrom)
	}
	if cfg.ResendTo != "frodi.carla@gmail.com" {
		t.Errorf("ResendTo = %q", cfg.ResendTo)
	}
	wantOrigins := []string{"https://carlaochfrodi.wedding", "https://staging.carlaochfrodi.wedding"}
	if len(cfg.CORSAllowedOrigins) != len(wantOrigins) {
		t.Fatalf("CORSAllowedOrigins = %v, want %v", cfg.CORSAllowedOrigins, wantOrigins)
	}
	for i := range wantOrigins {
		if cfg.CORSAllowedOrigins[i] != wantOrigins[i] {
			t.Errorf("CORSAllowedOrigins[%d] = %q, want %q", i, cfg.CORSAllowedOrigins[i], wantOrigins[i])
		}
	}
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9090")
	}
	if cfg.SecureCookie {
		t.Errorf("SecureCookie = true, want false")
	}
}

func TestLoad_DefaultsPortAndDBPath(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "x")
	t.Setenv("SESSION_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("RESEND_API_KEY", "x")
	t.Setenv("RESEND_FROM", "x@x")
	t.Setenv("RESEND_TO", "x@x")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://example.com")
	t.Setenv("DB_PATH", "")
	t.Setenv("PORT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port default = %q, want 8080", cfg.Port)
	}
	if cfg.DBPath != "/data/wedding.db" {
		t.Errorf("DBPath default = %q, want /data/wedding.db", cfg.DBPath)
	}
	if !cfg.SecureCookie {
		t.Errorf("SecureCookie default = false, want true")
	}
}

func TestLoad_RejectsShortSessionSecret(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "x")
	t.Setenv("SESSION_SECRET", strings.Repeat("a", 31)) // one byte short
	t.Setenv("RESEND_API_KEY", "x")
	t.Setenv("RESEND_FROM", "x@x")
	t.Setenv("RESEND_TO", "x@x")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://example.com")
	t.Setenv("DB_PATH", "")
	t.Setenv("PORT", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should error when SESSION_SECRET is shorter than 32 bytes")
	}
	if !errors.Is(err, ErrWeakSessionSecret) {
		t.Errorf("error = %v, want ErrWeakSessionSecret", err)
	}
}

func TestLoad_AcceptsExactly32ByteSessionSecret(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "x")
	t.Setenv("SESSION_SECRET", strings.Repeat("a", 32))
	t.Setenv("RESEND_API_KEY", "x")
	t.Setenv("RESEND_FROM", "x@x")
	t.Setenv("RESEND_TO", "x@x")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://example.com")
	t.Setenv("DB_PATH", "")
	t.Setenv("PORT", "")

	if _, err := Load(); err != nil {
		t.Fatalf("Load() with 32-byte secret errored: %v", err)
	}
}

func TestLoad_RequiresAdminPassword(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "")
	t.Setenv("SESSION_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("RESEND_API_KEY", "x")
	t.Setenv("RESEND_FROM", "x@x")
	t.Setenv("RESEND_TO", "x@x")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://example.com")
	t.Setenv("DB_PATH", "")
	t.Setenv("PORT", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should error when ADMIN_PASSWORD is empty")
	}
}
