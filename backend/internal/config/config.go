// Package config loads settings from environment variables.
package config

import (
	"errors"
	"os"
	"strconv"
)

// Config is the application configuration.
type Config struct {
	Addr             string
	DatabaseURL      string
	TelegramBotToken string
	TelegramChatID   string
	// UploadDir stores product and banner images served under /uploads/.
	UploadDir string
	// CookieSecure marks the admin session cookie Secure; enable behind HTTPS.
	CookieSecure bool
	// TrustProxy uses X-Forwarded-For for the client IP (only behind a reverse proxy).
	TrustProxy bool
	// StaticDir is the Vite build (web/dist) served for all non-API routes; empty disables it.
	StaticDir string
	// PublicURL is the site origin used in canonical links, Open Graph tags and the sitemap.
	PublicURL string
}

// Load reads the configuration. DATABASE_URL is required.
func Load() (Config, error) {
	c := Config{
		Addr:             getenv("APP_ADDR", ":8080"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID:   os.Getenv("TELEGRAM_CHAT_ID"),
		UploadDir:        getenv("UPLOAD_DIR", "uploads"),
		CookieSecure:     getbool("COOKIE_SECURE"),
		TrustProxy:       getbool("TRUST_PROXY"),
		StaticDir:        os.Getenv("STATIC_DIR"),
		PublicURL:        getenv("PUBLIC_URL", "http://localhost:8080"),
	}
	if c.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	return c, nil
}

// TelegramEnabled reports whether order notifications are configured.
func (c Config) TelegramEnabled() bool {
	return c.TelegramBotToken != "" && c.TelegramChatID != ""
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getbool(key string) bool {
	v, _ := strconv.ParseBool(os.Getenv(key))
	return v
}
