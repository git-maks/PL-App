// Package config reads runtime configuration from environment variables (12-factor).
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr        string // listen address, e.g. ":8080"
	DatabaseURL string
	// BaseURL is the public origin + base path, e.g. "https://uniquest.pollub.pl". Used in QR codes and CSRF checks.
	BaseURL string
	// CookieSecure enables the Secure flag and the __Host- cookie prefix (disable only for plain-http local dev).
	CookieSecure bool
	SessionTTL   time.Duration
	// DevLoginCodes returns login codes in the API response instead of e-mailing them (development/demo only).
	DevLoginCodes bool
	// OpenAllStations treats every station's scan window as open (local testing only).
	OpenAllStations bool
	SMTPHost        string
	SMTPPort        int
	SMTPUser        string
	SMTPPassword    string
	SMTPFrom        string
	VAPIDPublic     string
	VAPIDPrivate    string
	VAPIDSubject    string
	LogLevel        string
	Version         string
	TrustProxy      bool // honour X-Forwarded-For from the reverse proxy (Caddy)
	// RateLimits can be switched off for load-test environments only.
	RateLimits bool
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func boolEnv(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	return err == nil && b
}

func Load() (Config, error) {
	c := Config{
		Addr:            env("UQ_ADDR", ":8080"),
		DatabaseURL:     env("DATABASE_URL", "postgres://uniquest@127.0.0.1:5432/uniquest?sslmode=disable"),
		BaseURL:         strings.TrimRight(env("UQ_BASE_URL", "http://localhost:8080"), "/"),
		CookieSecure:    boolEnv("UQ_COOKIE_SECURE", true),
		DevLoginCodes:   boolEnv("UQ_DEV_LOGIN_CODES", false),
		OpenAllStations: boolEnv("UQ_OPEN_ALL_STATIONS", false),
		SMTPHost:        env("UQ_SMTP_HOST", ""),
		SMTPUser:        env("UQ_SMTP_USER", ""),
		SMTPPassword:    env("UQ_SMTP_PASSWORD", ""),
		SMTPFrom:        env("UQ_SMTP_FROM", "UniQuest <no-reply@localhost>"),
		VAPIDPublic:     env("UQ_VAPID_PUBLIC", ""),
		VAPIDPrivate:    env("UQ_VAPID_PRIVATE", ""),
		VAPIDSubject:    env("UQ_VAPID_SUBJECT", "mailto:uniquest@pollub.pl"),
		LogLevel:        env("UQ_LOG_LEVEL", "info"),
		Version:         env("UQ_VERSION", "dev"),
		TrustProxy:      boolEnv("UQ_TRUST_PROXY", false),
		RateLimits:      boolEnv("UQ_RATE_LIMITS", true),
	}
	port, err := strconv.Atoi(env("UQ_SMTP_PORT", "587"))
	if err != nil {
		return c, fmt.Errorf("UQ_SMTP_PORT: %w", err)
	}
	c.SMTPPort = port
	ttl, err := time.ParseDuration(env("UQ_SESSION_TTL", "720h"))
	if err != nil {
		return c, fmt.Errorf("UQ_SESSION_TTL: %w", err)
	}
	c.SessionTTL = ttl
	if _, err := url.Parse(c.BaseURL); err != nil {
		return c, fmt.Errorf("UQ_BASE_URL: %w", err)
	}
	return c, nil
}

// Origin returns scheme://host of BaseURL (for Origin header checks).
func (c Config) Origin() string {
	u, _ := url.Parse(c.BaseURL)
	return u.Scheme + "://" + u.Host
}

// BasePath returns the path part of BaseURL ("" or "/something").
func (c Config) BasePath() string {
	u, _ := url.Parse(c.BaseURL)
	return strings.TrimRight(u.Path, "/")
}

// SessionCookie is the cookie name; the __Host- prefix requires Secure.
func (c Config) SessionCookie() string {
	if c.CookieSecure {
		return "__Host-uq_session"
	}
	return "uq_session"
}
