// Package providers defines the IProxyProvider interface and provider registry
// for proxy-service to abstract away different proxy suppliers.
package providers

import (
	"context"
	"fmt"
	"sync"
)

// PurchaseRequest is the input to Purchase.
type PurchaseRequest struct {
	ProductID string
	Quantity  int
	OrderID   string
}

// PurchaseResult is the output of Purchase.
type PurchaseResult struct {
	ProviderOrderID string
	Credentials     map[string]any // proxies, username, password, etc.
}

// IProxyProvider defines the contract all proxy providers must implement.
type IProxyProvider interface {
	// Name returns the unique adapter name (matches adapter_type in DB).
	Name() string

	// Purchase buys proxies from the provider and returns credentials.
	Purchase(ctx context.Context, req PurchaseRequest) (*PurchaseResult, error)

	// GetUsage returns current bandwidth/connection usage for an order.
	GetUsage(ctx context.Context, providerOrderID string) (map[string]any, error)

	// Cancel cancels an active order at the provider.
	Cancel(ctx context.Context, providerOrderID string) error

	// HealthCheck verifies the provider API is reachable.
	HealthCheck(ctx context.Context) error
}

// Registry is a thread-safe map of provider name → provider ID → IProxyProvider.
type Registry struct {
	mu        sync.RWMutex
	byName    map[string]IProxyProvider // key: adapter_type
	byID      map[string]IProxyProvider // key: provider UUID string
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		byName: make(map[string]IProxyProvider),
		byID:   make(map[string]IProxyProvider),
	}
}

// Register adds a provider to the registry.
func (r *Registry) Register(providerID string, p IProxyProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byName[p.Name()] = p
	r.byID[providerID] = p
}

// Get returns a provider by UUID string.
func (r *Registry) Get(providerID string) IProxyProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byID[providerID]
}

// GetByName returns a provider by adapter type name.
func (r *Registry) GetByName(name string) (IProxyProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.byName[name]
	if !ok {
		return nil, fmt.Errorf("providers.Registry: provider %q not found", name)
	}
	return p, nil
}

// List returns all registered providers.
func (r *Registry) List() []IProxyProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]IProxyProvider, 0, len(r.byName))
	for _, p := range r.byName {
		result = append(result, p)
	}
	return result
}
