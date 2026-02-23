package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config — конфигурация приложения, загружаемая из переменных окружения.
type Config struct {
	// Telegram
	TelegramToken  string
	AllowedChatIDs []int64

	// LLM (Gemini)
	GeminiAPIKey string
	GeminiModel  string // По умолчанию "gemini-2.0-flash"

	// PostgreSQL
	DatabaseURL string

	// HTTP Server
	Port string // По умолчанию "8080"
}

// Load загружает конфигурацию из переменных окружения.
func Load() (*Config, error) {
	cfg := &Config{
		TelegramToken: os.Getenv("TELEGRAM_TOKEN"),
		GeminiAPIKey:  os.Getenv("GEMINI_API_KEY"),
		GeminiModel:   os.Getenv("GEMINI_MODEL"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		Port:          os.Getenv("PORT"),
	}

	// Обязательные поля
	if cfg.TelegramToken == "" {
		return nil, fmt.Errorf("config: TELEGRAM_TOKEN is required")
	}
	if cfg.GeminiAPIKey == "" {
		return nil, fmt.Errorf("config: GEMINI_API_KEY is required")
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}

	// Значения по умолчанию
	if cfg.GeminiModel == "" {
		cfg.GeminiModel = "gemini-2.0-flash"
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	// Парсим список разрешённых Chat ID ("123,456,789")
	if raw := os.Getenv("ALLOWED_CHAT_IDS"); raw != "" {
		ids := strings.Split(raw, ",")
		for _, s := range ids {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			id, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("config: invalid chat ID %q: %w", s, err)
			}
			cfg.AllowedChatIDs = append(cfg.AllowedChatIDs, id)
		}
	}

	return cfg, nil
}
