package config

import (
	"fmt"
	"os"
)

// Config holds all configuration for billing-service.
type Config struct {
	Port        string
	Env         string
	LogLevel    string
	DatabaseURL string
	NatsURL     string

	// Stripe
	StripeSecretKey      string
	StripeWebhookSecret  string

	// VNPay
	VNPayTMNCode    string
	VNPayHashSecret string

	// MoMo
	MoMoPartnerCode string
	MoMoAccessKey   string
	MoMoSecretKey   string

	// App
	FrontendBaseURL string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:            getEnv("PORT", "8085"),
		Env:             getEnv("ENV", "development"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		DatabaseURL:     requireEnv("DATABASE_URL"),
		NatsURL:         getEnv("NATS_URL", "nats://localhost:4222"),
		StripeSecretKey: getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		VNPayTMNCode:    getEnv("VNPAY_TMN_CODE", ""),
		VNPayHashSecret: getEnv("VNPAY_HASH_SECRET", ""),
		FrontendBaseURL: getEnv("FRONTEND_BASE_URL", "http://localhost:3000"),
	}
	return cfg, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("config: required env var %q is not set", key))
	}
	return v
}
