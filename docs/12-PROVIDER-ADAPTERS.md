# Provider Adapters — Infrastructure Design

**Document ID**: PVP-DOC-012  
**Version**: 1.0.0  
**Component**: infrastructure/providers  

---

## 1. Overview

Provider Adapters implement interface `IProxyProvider` thống nhất cho tất cả proxy providers. Mỗi provider có API khác nhau — adapter chuyển đổi sang format chuẩn của hệ thống.

---

## 2. IProxyProvider Interface

```go
type IProxyProvider interface {
    Name() string
    
    Purchase(ctx context.Context, req PurchaseRequest) (*PurchaseResult, error)
    GetCredentials(ctx context.Context, providerOrderID string) (*Credentials, error)
    CheckOrderStatus(ctx context.Context, providerOrderID string) (*OrderStatus, error)
    Extend(ctx context.Context, providerOrderID string, req ExtendRequest) error
    Cancel(ctx context.Context, providerOrderID string) error
    
    GetBalance(ctx context.Context) (*ProviderBalance, error)
    GetProducts(ctx context.Context) ([]ProviderProduct, error)
    
    // Health check
    Ping(ctx context.Context) error
}
```

### Standard Types

```go
type PurchaseRequest struct {
    ProductCode string
    Quantity    int
    Duration    int    // days, 0 for bandwidth-based
    Location    string // country code
    Metadata    map[string]string
}

type PurchaseResult struct {
    ProviderOrderID string
    Status          string  // active|pending|processing
    Credentials     *Credentials
    ExpiresAt       *time.Time
    BandwidthGB     *float64
}

type Credentials struct {
    Host     string
    Port     int
    Username string
    Password string
    Protocol string  // http|socks5|https
}
```

---

## 3. Provider Registry

```go
type ProviderRegistry struct {
    providers map[string]IProxyProvider
    mu        sync.RWMutex
}

func (r *ProviderRegistry) Register(p IProxyProvider) {
    r.mu.Lock()
    r.providers[p.Name()] = p
    r.mu.Unlock()
}

func (r *ProviderRegistry) Get(name string) (IProxyProvider, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    p, ok := r.providers[name]
    return p, ok
}
```

---

## 4. Provider Router

```go
type ProviderRouter struct {
    registry *ProviderRegistry
    db       *ProviderRepository  // active providers + priority from DB
}

func (r *ProviderRouter) Route(req PurchaseRequest) (IProxyProvider, error) {
    // Load active providers supporting this request
    candidates := r.db.FindProviders(ProviderFilter{
        ProxyType: req.ProxyType,
        Protocol:  req.Protocol,
        Location:  req.Location,
        IsActive:  true,
    })
    
    // Sort by priority (configurable in admin)
    sort.Slice(candidates, func(i, j int) bool {
        return candidates[i].Priority > candidates[j].Priority
    })
    
    // Try each in order
    for _, candidate := range candidates {
        provider, ok := r.registry.Get(candidate.Name)
        if !ok { continue }
        if err := provider.Ping(ctx); err != nil {
            log.Warn("provider unavailable", "provider", candidate.Name)
            continue
        }
        return provider, nil
    }
    
    return nil, ErrNoAvailableProvider
}
```

---

## 5. Adding a New Provider

1. Create file: `infrastructure/providers/{provider_name}/adapter.go`
2. Implement `IProxyProvider` interface
3. Register in `providers/registry.go`
4. Add provider config to database (`proxy.providers` table)
5. Add environment variables for API credentials
6. Write unit tests với mock HTTP server
7. Document in this file (section 6)

---

## 6. Provider Implementation Notes

### Generic HTTP Adapter Pattern

```go
type GenericHTTPProvider struct {
    name    string
    baseURL string
    apiKey  string
    client  *http.Client  // với timeout + retry transport
}

// Retry Transport: 3 retries, exponential backoff 1s/2s/4s
// Timeout per request: 30s
// Circuit breaker: open after 5 failures in 60s
```

### Error Mapping

Mỗi adapter phải map provider-specific errors sang error types chuẩn:

```go
var (
    ErrInsufficientStock  = errors.New("provider: insufficient stock")
    ErrInvalidLocation    = errors.New("provider: location not supported")  
    ErrProviderBalance    = errors.New("provider: insufficient balance")
    ErrOrderNotFound      = errors.New("provider: order not found")
    ErrProviderUnavailable = errors.New("provider: service unavailable")
)
```

---

## 7. Configuration Per Provider

Provider configs stored in `proxy.providers.config` (JSONB, encrypted):

```json
{
  "api_key": "encrypted:...",
  "api_secret": "encrypted:...",
  "base_url": "https://api.provider.com/v1",
  "timeout_seconds": 30,
  "retry_count": 3,
  "rate_limit_per_minute": 60
}
```

---

## 8. Testing Providers

```bash
# Test a specific provider
make test-provider PROVIDER=smartproxy

# Test all providers (uses mock servers)
make test-providers

# Integration test (requires real credentials)
make test-provider-integration PROVIDER=smartproxy
```
