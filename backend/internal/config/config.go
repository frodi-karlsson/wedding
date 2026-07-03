package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	DBPath             string
	AdminPassword      string
	SessionSecret      string
	ResendAPIKey       string
	ResendFrom         string
	ResendTo           string
	CORSAllowedOrigins []string
	Port               string
	SecureCookie       bool
}

// ErrMissingEnvVars is returned when one or more required environment variables are absent.
var ErrMissingEnvVars = errors.New("missing required env vars")

// ErrWeakSessionSecret is returned when SESSION_SECRET is shorter than the
// minimum required length.
var ErrWeakSessionSecret = errors.New("SESSION_SECRET must be at least 32 bytes")

// minSessionSecretLen is the minimum acceptable SESSION_SECRET length in bytes.
// The session cookie is HMAC-SHA256 signed, so a secret of at least the hash's
// block/output size resists brute forcing.
const minSessionSecretLen = 32

// Load reads configuration from environment variables. Required vars are
// ADMIN_PASSWORD, SESSION_SECRET, RESEND_API_KEY, RESEND_FROM, RESEND_TO,
// and CORS_ALLOWED_ORIGINS. DB_PATH defaults to /data/wedding.db and PORT
// defaults to 8080.
func Load() (Config, error) {
	cfg := Config{
		DBPath:             envOr("DB_PATH", "/data/wedding.db"),
		AdminPassword:      os.Getenv("ADMIN_PASSWORD"),
		SessionSecret:      os.Getenv("SESSION_SECRET"),
		ResendAPIKey:       os.Getenv("RESEND_API_KEY"),
		ResendFrom:         os.Getenv("RESEND_FROM"),
		ResendTo:           os.Getenv("RESEND_TO"),
		CORSAllowedOrigins: splitCSV(os.Getenv("CORS_ALLOWED_ORIGINS")),
		Port:               envOr("PORT", "8080"),
		SecureCookie:       envOrBool("SECURE_COOKIE", true),
	}

	var missing []string
	if cfg.AdminPassword == "" {
		missing = append(missing, "ADMIN_PASSWORD")
	}
	if cfg.SessionSecret == "" {
		missing = append(missing, "SESSION_SECRET")
	}
	if cfg.ResendAPIKey == "" {
		missing = append(missing, "RESEND_API_KEY")
	}
	if cfg.ResendFrom == "" {
		missing = append(missing, "RESEND_FROM")
	}
	if cfg.ResendTo == "" {
		missing = append(missing, "RESEND_TO")
	}
	if len(cfg.CORSAllowedOrigins) == 0 {
		missing = append(missing, "CORS_ALLOWED_ORIGINS")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("%w: %s", ErrMissingEnvVars, strings.Join(missing, ", "))
	}
	// Guard against a weak session secret: a short secret makes the HMAC-signed
	// session cookie brute-forceable. Only checked once the var is present.
	if len(cfg.SessionSecret) < minSessionSecretLen {
		return Config{}, fmt.Errorf("%w: got %d bytes", ErrWeakSessionSecret, len(cfg.SessionSecret))
	}
	return cfg, nil
}

func envOr(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func envOrBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
