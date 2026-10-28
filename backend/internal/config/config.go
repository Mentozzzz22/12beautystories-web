// Package config loads settings from environment variables.
package config

import (
	"errors"
	"os"
)

// Config is the application configuration.
type Config struct {
	Addr             string
	DatabaseURL      string
	TelegramBotToken string
	TelegramChatID   string
}

// Load reads the configuration. DATABASE_URL is required.
func Load() (Config, error) {
	c := Config{
		Addr:             getenv("APP_ADDR", ":8080"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID:   os.Getenv("TELEGRAM_CHAT_ID"),
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
