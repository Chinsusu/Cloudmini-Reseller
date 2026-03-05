package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ProxmoxNodeConfig holds config for a single Proxmox node.
type ProxmoxNodeConfig struct {
	Name        string
	Host        string
	Port        int
	TokenID     string
	TokenSecret string
	VerifyCert  bool
}

// Config holds all configuration for vps-service.
type Config struct {
	Port        string
	Env         string
	LogLevel    string
	DatabaseURL string
	NatsURL     string
	EncryptionKey string

	// Proxmox nodes (comma-separated -> parsed)
	ProxmoxNodes []ProxmoxNodeConfig

	// Billing internal URL
	BillingServiceURL string

	// Next VMID starting value (should match Proxmox DC settings)
	VMIDStart int
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:              getEnv("PORT", "8083"),
		Env:               getEnv("ENV", "development"),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		DatabaseURL:       requireEnv("DATABASE_URL"),
		NatsURL:           getEnv("NATS_URL", "nats://localhost:4222"),
		EncryptionKey:     requireEnv("ENCRYPTION_KEY"),
		BillingServiceURL: getEnv("BILLING_SERVICE_URL", "http://localhost:8085"),
		VMIDStart:         1000,
	}

	// Parse Proxmox nodes from env:
	// PROXMOX_NODES=pve1:pve1.example.com:8006:root@pam!token:secret,pve2:...
	nodesRaw := os.Getenv("PROXMOX_NODES")
	if nodesRaw != "" {
		for _, part := range strings.Split(nodesRaw, ",") {
			fields := strings.Split(strings.TrimSpace(part), ":")
			if len(fields) < 5 {
				return nil, fmt.Errorf("config: malformed PROXMOX_NODES entry: %q", part)
			}
			port, _ := strconv.Atoi(fields[2])
			if port == 0 {
				port = 8006
			}
			cfg.ProxmoxNodes = append(cfg.ProxmoxNodes, ProxmoxNodeConfig{
				Name:        fields[0],
				Host:        fields[1],
				Port:        port,
				TokenID:     fields[3],
				TokenSecret: fields[4],
				VerifyCert:  false, // set true in production
			})
		}
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
		panic(fmt.Sprintf("required env var not set: %s", key))
	}
	return v
}
