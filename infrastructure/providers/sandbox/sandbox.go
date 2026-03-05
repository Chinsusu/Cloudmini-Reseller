// Package sandbox provides a mock/sandbox IProxyProvider implementation
// for local development and testing without real provider credentials.
package sandbox

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/pvp/providers"
)

// SandboxProvider is a mock provider that simulates proxy purchase.
type SandboxProvider struct{}

// New creates a SandboxProvider.
func New() *SandboxProvider { return &SandboxProvider{} }

// Name returns the adapter type identifier.
func (p *SandboxProvider) Name() string { return "sandbox" }

// Purchase simulates buying proxies (no real HTTP call).
func (p *SandboxProvider) Purchase(_ context.Context, req providers.PurchaseRequest) (*providers.PurchaseResult, error) {
	// Simulate occasional failures (10% of the time in sandbox)
	if rand.Intn(10) == 0 {
		return nil, fmt.Errorf("sandbox: simulated provider failure")
	}

	// Generate fake proxy credentials
	orderID := fmt.Sprintf("SANDBOX-%d", time.Now().UnixNano())
	proxyList := make([]string, req.Quantity)
	for i := range proxyList {
		proxyList[i] = fmt.Sprintf("192.168.%d.%d:3128", rand.Intn(255), rand.Intn(255))
	}

	return &providers.PurchaseResult{
		ProviderOrderID: orderID,
		Credentials: map[string]any{
			"proxies":   proxyList,
			"username":  "sandbox_user",
			"password":  "sandbox_pass",
			"protocol":  "http",
			"order_id":  req.OrderID,
			"issued_at": time.Now().UTC().Format(time.RFC3339),
		},
	}, nil
}

// GetUsage returns fake usage statistics.
func (p *SandboxProvider) GetUsage(_ context.Context, providerOrderID string) (map[string]any, error) {
	return map[string]any{
		"provider_order_id": providerOrderID,
		"bandwidth_used_gb": rand.Float64() * 10,
		"bandwidth_limit_gb": 100.0,
		"requests_count":    rand.Intn(50000),
	}, nil
}

// Cancel simulates order cancellation.
func (p *SandboxProvider) Cancel(_ context.Context, providerOrderID string) error {
	return nil
}

// HealthCheck always succeeds for sandbox.
func (p *SandboxProvider) HealthCheck(_ context.Context) error { return nil }
