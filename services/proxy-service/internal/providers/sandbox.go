// Package providers — sandbox adapter for development/testing.
// Returns realistic-looking fake proxy credentials without calling any real provider.
package providers

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// mockIPs are realistic looking datacenter IPs for sandbox testing.
var mockIPs = []string{
	"45.77.34.182", "103.124.106.55", "188.166.72.13",
	"165.22.110.47", "157.230.84.121", "104.248.55.12",
	"128.199.213.44", "167.172.95.33", "142.93.118.76",
	"178.62.194.50", "159.89.45.111", "68.183.77.202",
}

// SandboxAdapter is a fake provider that returns realistic-looking credentials.
// Used in development and integration testing.
type SandboxAdapter struct{}

// NewSandboxAdapter creates a sandbox adapter.
func NewSandboxAdapter() *SandboxAdapter { return &SandboxAdapter{} }

// Purchase returns fake proxy credentials that look realistic.
func (s *SandboxAdapter) Purchase(ctx context.Context, req PurchaseRequest) (*PurchaseResult, error) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	creds := make([]ProxyCredential, req.Quantity)
	for i := range creds {
		ip := mockIPs[rng.Intn(len(mockIPs))]
		port := 10000 + rng.Intn(55000)
		user := fmt.Sprintf("pvp_%06x", rng.Int31())
		pass := fmt.Sprintf("%x%x", rng.Int63(), rng.Int31())
		creds[i] = ProxyCredential{
			Host:     ip,
			Port:     port,
			Username: user,
			Password: pass,
			Protocol: "http",
			Country:  "US",
		}
	}
	return &PurchaseResult{
		ProviderOrderID: fmt.Sprintf("sandbox-%d-%04x", time.Now().UnixNano(), rng.Int31n(0xFFFF)),
		Credentials:     creds,
	}, nil
}

// Cancel is a no-op for sandbox.
func (s *SandboxAdapter) Cancel(_ context.Context, _ string) error { return nil }

// Suspend is a no-op for sandbox.
func (s *SandboxAdapter) Suspend(_ context.Context, _ string) error { return nil }

// Resume is a no-op for sandbox.
func (s *SandboxAdapter) Resume(_ context.Context, _ string) error { return nil }

// CheckStatus always returns "active" for sandbox.
func (s *SandboxAdapter) CheckStatus(_ context.Context, _ string) (string, error) {
	return "active", nil
}
