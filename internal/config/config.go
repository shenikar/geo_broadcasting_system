package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config - структура для хранения конфигурации приложения
type Config struct {
	DatabaseURL string `env:"DATABASE_URL"`
	HTTPPort    string `env:"HTTP_PORT" envDefault:"8080"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`

	// Redis Config
	RedisAddr string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	RedisPass string `env:"REDIS_PASSWORD"`
	RedisDB   int    `env:"REDIS_DB" envDefault:"0"`

	// Webhook Config
	WebhookURL     string        `env:"WEBHOOK_URL"`
	WebhookSecret  string        `env:"WEBHOOK_SECRET"`
	WebhookTimeout time.Duration `env:"WEBHOOK_TIMEOUT" envDefault:"5s"`

	// Stats Config
	StatsTimeWindowMinutes int `env:"STATS_TIME_WINDOW_MINUTES" envDefault:"60"`

	// API Keys for authentication
	APIKeys []string `env:"API_KEYS"`
}

// LoadConfig загружает конфигурацию из переменных окружения и .env файла
func LoadConfig() (*Config, error) {
	// Загрузка переменных окружения из .env файла (если есть)
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("ошибка загрузки файла .env: %w", err)
	}

	cfg := &Config{
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		HTTPPort:               getEnv("HTTP_PORT", "8080"),
		LogLevel:               getEnv("LOG_LEVEL", "info"),
		RedisAddr:              getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass:              os.Getenv("REDIS_PASSWORD"),
		RedisDB:                getEnvAsInt("REDIS_DB", 0),
		WebhookURL:             os.Getenv("WEBHOOK_URL"),
		WebhookSecret:          os.Getenv("WEBHOOK_SECRET"),
		WebhookTimeout:         getEnvAsDuration("WEBHOOK_TIMEOUT", 5*time.Second),
		StatsTimeWindowMinutes: getEnvAsInt("STATS_TIME_WINDOW_MINUTES", 60),
	}

	// Загрузка API ключей
	apiKeysStr := os.Getenv("API_KEYS")
	if apiKeysStr != "" {
		cfg.APIKeys = strings.Split(apiKeysStr, ",")
		for i, key := range cfg.APIKeys {
			cfg.APIKeys[i] = strings.TrimSpace(key)
		}
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	return cfg, nil
}

// getEnv возвращает значение переменной окружения или значение по умолчанию
func getEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsInt возвращает значение переменной окружения как int или значение по умолчанию
func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsDuration возвращает значение переменной окружения как time.Duration или значение по умолчанию
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if durationValue, err := time.ParseDuration(value); err == nil {
			return durationValue
		}
	}
	return defaultValue
}
