// Package providers is the internal proxy-service provider registry.
// It bridges the infrastructure/providers module (IProxyProvider interface)
// to the proxy-service usecase layer using a type-safe Registry.
//
// The Registry maps provider IDs → IProxyProvider implementations.
// In main.go, concrete adapters (sandbox, vendor, etc.) are registered.
package providers

import "context"

// PurchaseRequest is the request to purchase proxies from a provider.
type PurchaseRequest struct {
	ProductID string // provider's product/plan ID
	Quantity  int    // number of proxy units
	OrderID   string // our internal order ID for reference
	Country   string // optional country filter
	Protocol  string // http|socks5|etc
}

// PurchaseResult is the result returned by the provider.
type PurchaseResult struct {
	ProviderOrderID string            // provider's reference ID
	Credentials     []ProxyCredential // list of proxies purchased
}

// ProxyCredential represents a single proxy credential.
type ProxyCredential struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Protocol string `json:"protocol"`
	Country  string `json:"country,omitempty"`
}

// IProxyProvider is the interface every proxy provider adapter must implement.
type IProxyProvider interface {
	// Purchase buys proxy units from the provider.
	Purchase(ctx context.Context, req PurchaseRequest) (*PurchaseResult, error)
	// Cancel cancels/releases a previously purchased order.
	Cancel(ctx context.Context, providerOrderID string) error
	// CheckStatus checks the status of a provider order.
	CheckStatus(ctx context.Context, providerOrderID string) (string, error)
}

// Registry maps provider IDs to IProxyProvider implementations.
// Thread-safe for reads after initial setup (register all providers at startup).
type Registry struct {
	providers map[string]IProxyProvider
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]IProxyProvider)}
}

// Register adds a provider adapter to the registry.
func (r *Registry) Register(providerID string, p IProxyProvider) {
	r.providers[providerID] = p
}

// Get returns the provider for a given ID, or nil if not found.
func (r *Registry) Get(providerID string) IProxyProvider {
	return r.providers[providerID]
}

// All returns all registered provider IDs.
func (r *Registry) All() []string {
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	return ids
}
