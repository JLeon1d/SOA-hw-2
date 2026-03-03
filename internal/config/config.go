package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Database  DatabaseConfig
	JWT       JWTConfig
	Server    ServerConfig
	RateLimit RateLimitConfig
	Log       LogConfig
}

type DatabaseConfig struct {
	URL string
}

type JWTConfig struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

type ServerConfig struct {
	Port string
}

type RateLimitConfig struct {
	Minutes int
}

type LogConfig struct {
	Level string
}

func Load() (*Config, error) {
	cfg := &Config{
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", "postgres://marketplace:marketplace_password@localhost:5432/marketplace?sslmode=disable"),
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			AccessExpiry:  parseDuration(getEnv("JWT_ACCESS_EXPIRY", "30m"), 30*time.Minute),
			RefreshExpiry: parseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h"), 168*time.Hour),
		},
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
		},
		RateLimit: RateLimitConfig{
			Minutes: parseInt(getEnv("RATE_LIMIT_MINUTES", "5"), 5),
		},
		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
	}

	if cfg.Database.URL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(value string, defaultValue time.Duration) time.Duration {
	if d, err := time.ParseDuration(value); err == nil {
		return d
	}
	return defaultValue
}

func parseInt(value string, defaultValue int) int {
	if i, err := strconv.Atoi(value); err == nil {
		return i
	}
	return defaultValue
}
