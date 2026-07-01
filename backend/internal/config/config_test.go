package config

import (
	"testing"
)

func TestLoad_ReadsEnvVars(t *testing.T) {
	t.Setenv("DB_PATH", "/data/wedding.db")
	t.Setenv("ADMIN_PASSWORD", "secret-pass")
	t.Setenv("SESSION_SECRET", "session-secret")
	t.Setenv("RESEND_API_KEY", "re_test_key")
	t.Setenv("RESEND_FROM", "rsvp@carlaochfrodi.wedding")
	t.Setenv("RESEND_TO", "frodi.carla@gmail.com")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://carlaochfrodi.wedding,https://staging.carlaochfrodi.wedding")
	t.Setenv("PORT", "9090")

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
	if cfg.SessionSecret != "session-secret" {
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
}

func TestLoad_DefaultsPortAndDBPath(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "x")
	t.Setenv("SESSION_SECRET", "x")
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
}

func TestLoad_RequiresAdminPassword(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "")
	t.Setenv("SESSION_SECRET", "x")
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
