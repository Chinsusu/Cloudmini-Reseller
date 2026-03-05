package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds all configuration for user-service.
type Config struct {
	// Server
	Port string
	Env  string

	// Database
	DatabaseURL string

	// NATS
	NatsURL string

	// Redis
	RedisURL string

	// JWT
	JWTSecret        []byte
	JWTAccessTTL     time.Duration
	JWTRefreshTTL    time.Duration
	JWTAdminRefreshTTL time.Duration

	// App
	LogLevel string

	// Rate limiting
	MaxSessionsPerUser int
	MaxLoginAttempts   int
	LockoutDuration    time.Duration
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Port:               getEnv("PORT", "8081"),
		Env:                getEnv("ENV", "development"),
		DatabaseURL:        requireEnv("DATABASE_URL"),
		NatsURL:            getEnv("NATS_URL", "nats://localhost:4222"),
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		MaxSessionsPerUser: 5,
		MaxLoginAttempts:   10,
		LockoutDuration:    15 * time.Minute,
	}

	secret := requireEnv("JWT_SECRET")
	if len(secret) < 32 {
		return nil, fmt.Errorf("config: JWT_SECRET must be at least 32 characters")
	}
	cfg.JWTSecret = []byte(secret)

	var err error
	cfg.JWTAccessTTL, err = parseDuration("JWT_ACCESS_TTL", "15m")
	if err != nil {
		return nil, err
	}
	cfg.JWTRefreshTTL, err = parseDuration("JWT_REFRESH_TTL", "168h")
	if err != nil {
		return nil, err
	}
	cfg.JWTAdminRefreshTTL, err = parseDuration("JWT_ADMIN_REFRESH_TTL", "720h")
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func requireEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("config: required environment variable %q is not set", key))
	}
	return val
}

func parseDuration(key, defaultVal string) (time.Duration, error) {
	val := getEnv(key, defaultVal)
	d, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("config: invalid duration for %s: %w", key, err)
	}
	return d, nil
}
