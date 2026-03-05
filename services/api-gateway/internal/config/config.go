package config

import (
	"fmt"
	"os"
)

// Config holds all configuration for api-gateway.
type Config struct {
	Port     string
	Env      string
	LogLevel string

	// JWT (for validation — same secret as user-service)
	JWTSecret []byte

	// Redis (rate limiting)
	RedisURL string

	// Upstream service URLs
	UserServiceURL         string
	ProxyServiceURL        string
	VPSServiceURL          string
	ResellerServiceURL     string
	BillingServiceURL      string
	LogServiceURL          string
	NotificationServiceURL string

	// Rate limit (requests per minute)
	RateLimitAnonymous int
	RateLimitUser      int
	RateLimitReseller  int
	RateLimitAdmin     int
}

func Load() (*Config, error) {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		return nil, fmt.Errorf("config: JWT_SECRET must be at least 32 characters")
	}

	return &Config{
		Port:                   getEnv("PORT", "8080"),
		Env:                    getEnv("ENV", "development"),
		LogLevel:               getEnv("LOG_LEVEL", "info"),
		JWTSecret:              []byte(secret),
		RedisURL:               getEnv("REDIS_URL", "redis://localhost:6379"),
		UserServiceURL:         getEnv("USER_SERVICE_URL", "http://localhost:8081"),
		ProxyServiceURL:        getEnv("PROXY_SERVICE_URL", "http://localhost:8082"),
		VPSServiceURL:          getEnv("VPS_SERVICE_URL", "http://localhost:8083"),
		ResellerServiceURL:     getEnv("RESELLER_SERVICE_URL", "http://localhost:8084"),
		BillingServiceURL:      getEnv("BILLING_SERVICE_URL", "http://localhost:8085"),
		LogServiceURL:          getEnv("LOG_SERVICE_URL", "http://localhost:8087"),
		RateLimitAnonymous:     20,
		RateLimitUser:          120,
		RateLimitReseller:      500,
		RateLimitAdmin:         2000,
	}, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
