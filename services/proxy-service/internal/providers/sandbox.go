// Package providers — sandbox adapter for development/testing.
// Returns fake proxy credentials without calling any real provider.
package providers

import (
	"context"
	"fmt"
	"time"
)

// SandboxAdapter is a fake provider that returns hardcoded credentials.
// Used in development and integration testing.
type SandboxAdapter struct{}

// NewSandboxAdapter creates a sandbox adapter.
func NewSandboxAdapter() *SandboxAdapter { return &SandboxAdapter{} }

// Purchase returns fake proxy credentials.
func (s *SandboxAdapter) Purchase(ctx context.Context, req PurchaseRequest) (*PurchaseResult, error) {
	creds := make([]ProxyCredential, req.Quantity)
	for i := range creds {
		creds[i] = ProxyCredential{
			Host:     "sandbox.proxy.pvp.io",
			Port:     10000 + i,
			Username: fmt.Sprintf("user_%d_%d", time.Now().UnixNano(), i),
			Password: "sandbox_pass",
			Protocol: "http",
			Country:  "US",
		}
	}
	return &PurchaseResult{
		ProviderOrderID: fmt.Sprintf("sandbox-%d", time.Now().UnixNano()),
		Credentials:     creds,
	}, nil
}

// Cancel is a no-op for sandbox.
func (s *SandboxAdapter) Cancel(_ context.Context, _ string) error { return nil }

// CheckStatus always returns "active" for sandbox.
func (s *SandboxAdapter) CheckStatus(_ context.Context, _ string) (string, error) {
	return "active", nil
}
