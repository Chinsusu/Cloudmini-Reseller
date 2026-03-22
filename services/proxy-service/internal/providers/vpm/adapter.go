package vpm

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/pvp/proxy-service/internal/providers"
)

// ProviderName is the adapter_type registered in the proxy.providers table.
const ProviderName = "vpm"

// Config holds VPM connection settings.
// Stored encrypted in proxy.providers.config JSONB.
type Config struct {
	BaseURL string `json:"base_url"` // e.g. "http://192.168.1.62:8080"
	APIKey  string `json:"api_key"`  // Bearer token from VPM /api-keys
}

// Adapter implements providers.IProxyProvider for the VPS Proxy Manager.
// VPM is a synchronous provider: POST /proxies returns credentials immediately.
type Adapter struct {
	client *Client
	cfg    Config
}

// NewAdapter creates a new VPM adapter.
func NewAdapter(cfg Config) *Adapter {
	return &Adapter{
		client: NewClient(cfg.BaseURL, cfg.APIKey),
		cfg:    cfg,
	}
}

// NewAdapterWithClient creates an adapter with an injected client (for testing).
func NewAdapterWithClient(cfg Config, client *Client) *Adapter {
	return &Adapter{client: client, cfg: cfg}
}

// Purchase calls POST /api/v1/proxies.
//
// Since VPM allocates proxies synchronously, this always returns
// a non-nil Credentials slice. The order_usecase will set status=active immediately.
//
// PurchaseRequest.Metadata keys:
//   - "ip_range_id"        — VPM IP range UUID (optional; VPM auto-selects if empty)
//   - "node_id"            — target node UUID  (optional)
//   - "bandwidth_limit_mb" — integer, 0 = unlimited (optional)
//   - "speed_limit_mbps"   — integer (optional)
//   - "auth_user"          — proxy username (optional; VPM generates if empty)
//   - "auth_pass"          — proxy password (optional; VPM generates if empty)
func (a *Adapter) Purchase(ctx context.Context, req providers.PurchaseRequest) (*providers.PurchaseResult, error) {
	m := req.Metadata
	if m == nil {
		m = map[string]string{}
	}

	createReq := CreateProxyRequest{
		Protocol:  req.Protocol,
		IPRangeID: m["ip_range_id"],
		NodeID:    m["node_id"],
		AuthUser:  m["auth_user"],
		AuthPass:  m["auth_pass"],
	}

	if s := m["bandwidth_limit_mb"]; s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			createReq.BandwidthLimitMB = v
		}
	}
	if s := m["speed_limit_mbps"]; s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			createReq.SpeedLimitMbps = v
		}
	}

	// Default protocol to socks5 if not specified
	if createReq.Protocol == "" {
		createReq.Protocol = "socks5"
	}

	summary, err := a.client.CreateProxy(ctx, createReq)
	if err != nil {
		return nil, mapError(err)
	}

	// VPM is synchronous — build credentials immediately.
	cred := providers.ProxyCredential{
		Host:     summary.Host,
		Port:     summary.Port,
		Username: summary.AuthUser,
		Password: summary.AuthPass,
		Protocol: summary.Protocol,
	}

	return &providers.PurchaseResult{
		ProviderOrderID: summary.ID, // VPM proxy UUID
		Credentials:     []providers.ProxyCredential{cred},
	}, nil
}

// Cancel calls DELETE /api/v1/proxies/{id}, permanently removing the proxy
// and releasing the port and IP allocation.
func (a *Adapter) Cancel(ctx context.Context, providerOrderID string) error {
	if err := a.client.DeleteProxy(ctx, providerOrderID); err != nil {
		return mapError(err)
	}
	return nil
}

// CheckStatus calls GET /api/v1/proxies/{id} and maps VPM status to Cloudmini constants.
//
// VPM status → Cloudmini:
//
//	"running"  → "active"
//	"stopped"  → "active"    (proxy is reserved; user can still re-start it)
//	"creating" → "processing"
//	"error"    → "failed"
func (a *Adapter) CheckStatus(ctx context.Context, providerOrderID string) (string, error) {
	proxy, err := a.client.GetProxy(ctx, providerOrderID)
	if err != nil {
		return "", mapError(err)
	}
	return mapStatus(proxy.Status), nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// mapStatus converts VPM proxy status strings to Cloudmini order status constants.
func mapStatus(s string) string {
	switch s {
	case "running", "stopped":
		return "active"
	case "creating":
		return "processing"
	case "error":
		return "failed"
	default:
		return "processing"
	}
}

// mapError converts VPM APIError to providers error types.
func mapError(err error) error {
	apiErr, ok := err.(*APIError)
	if !ok {
		return err
	}
	switch apiErr.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("%w: vpm auth failed: %s", providers.ErrProviderUnavailable, apiErr.Code)
	case http.StatusUnprocessableEntity, http.StatusBadRequest:
		return fmt.Errorf("%w: %s", providers.ErrInvalidConfig, apiErr.Message)
	case http.StatusNotFound:
		return fmt.Errorf("%w: proxy not found in vpm", providers.ErrProviderUnavailable)
	default:
		return fmt.Errorf("%w: %s", providers.ErrProviderUnavailable, apiErr.Message)
	}
}
