package vpm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/pvp/proxy-service/internal/providers"
)

// ProviderName is the adapter_type registered in the proxy.providers table.
const ProviderName = "vpm"

// Config holds VPM connection settings.
// Stored encrypted in proxy.providers.config JSONB.
type Config struct {
	BaseURL string `json:"base_url"` // e.g. "https://cz.resvn.net"
	APIKey  string `json:"api_key"`  // access_code from VPM
}

// Adapter implements providers.IProxyProvider for the VPS Proxy Manager (API v2).
// VPM is a synchronous provider: POST /api/v2/proxies returns credentials immediately.
//
// Protocol behaviour (v2):
//   - "default" → 2 proxy objects in response (HTTP + SOCKS5, same IP/auth)
//     ProviderOrderID stored as "http_id|socks_id"
//   - "http"    → 1 proxy object  (port_socks: null)
//   - "socks5"  → 1 proxy object  (port_http: null)
//
// Legacy v1 provider_order_ids (plain UUID, no "|") are handled transparently
// via fallback to v1 client methods in Cancel/Suspend/Resume/CheckStatus.
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

// ─── Purchase ─────────────────────────────────────────────────────────────────

// Purchase calls POST /api/v2/proxies and returns credentials synchronously.
//
// PurchaseRequest.Metadata keys (v2):
//   - "group_id"           — VPM region/group UUID
//   - "protocol"           — "default"|"http"|"socks5" (default: "default")
//   - "ipv4"               — specific IP (optional; VPM auto-selects if empty)
//   - "bandwidth_limit_mb" — integer string, 0 = unlimited
//   - "speed_limit_mbps"   — integer string
func (a *Adapter) Purchase(ctx context.Context, req providers.PurchaseRequest) (*providers.PurchaseResult, error) {
	m := req.Metadata
	if m == nil {
		m = map[string]string{}
	}

	protocol := m["protocol"]
	if protocol == "" {
		protocol = req.Protocol
	}
	if protocol == "" {
		protocol = "default"
	}

	createReq := CreateProxyV2Request{
		Protocol: protocol,
		GroupID:  m["group_id"],
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

	// Default: POST /api/v2/ipv4 with specific IP from metadata.
	// Region-based creation (POST /api/v2/proxies with group_id) is reserved for future use.
	ipv4 := m["ipv4"]
	var proxy *ProxySummaryV2
	var err error
	if ipv4 != "" {
		proxy, err = a.client.CreateProxyByIPV4(ctx, ipv4, protocol)
	} else {
		proxy, err = a.client.CreateProxyV2(ctx, createReq)
	}
	if err != nil {
		return nil, mapError(err)
	}
	if proxy == nil || proxy.ID == "" {
		return nil, fmt.Errorf("%w: vpm returned empty response", providers.ErrProviderUnavailable)
	}

	// Build credentials from single-object response.
	// protocol=default → both port_http and port_socks in same object → 2 credentials.
	// Other protocols (vmess/vless/ss/trojan/wireguard) → 1 credential using connection_string.
	var creds []providers.ProxyCredential
	switch proxy.Protocol {
	case "default":
		if proxy.PortHTTP > 0 {
			creds = append(creds, providers.ProxyCredential{
				Host:     proxy.IPv4,
				Port:     proxy.PortHTTP,
				Username: proxy.Username,
				Password: proxy.Password,
				Protocol: "http",
			})
		}
		if proxy.PortSOCKS > 0 {
			creds = append(creds, providers.ProxyCredential{
				Host:     proxy.IPv4,
				Port:     proxy.PortSOCKS,
				Username: proxy.Username,
				Password: proxy.Password,
				Protocol: "socks5",
			})
		}
	case "http":
		if proxy.PortHTTP > 0 {
			creds = append(creds, providers.ProxyCredential{
				Host:     proxy.IPv4,
				Port:     proxy.PortHTTP,
				Username: proxy.Username,
				Password: proxy.Password,
				Protocol: "http",
			})
		}
	case "socks5":
		if proxy.PortSOCKS > 0 {
			creds = append(creds, providers.ProxyCredential{
				Host:     proxy.IPv4,
				Port:     proxy.PortSOCKS,
				Username: proxy.Username,
				Password: proxy.Password,
				Protocol: "socks5",
			})
		}
	default:
		// vmess, vless, shadowsocks, trojan, wireguard — use connection_string
		port := proxy.effectivePort()
		creds = append(creds, providers.ProxyCredential{
			Host:             proxy.IPv4,
			Port:             port,
			Username:         proxy.Username,
			Password:         proxy.Password,
			Protocol:         proxy.Protocol,
			ConnectionString: proxy.ConnectionString,
		})
	}
	if len(creds) == 0 {
		return nil, fmt.Errorf("%w: vpm returned no usable credentials", providers.ErrProviderUnavailable)
	}

	return &providers.PurchaseResult{
		ProviderOrderID: proxy.ID,
		Credentials:     creds,
	}, nil
}

// ─── Cancel ───────────────────────────────────────────────────────────────────

// Cancel permanently deletes a proxy from VPM.
// DELETE /api/v2/ipv4/{id}?access_code=<key>
// A single DELETE removes both HTTP and SOCKS5 inbounds for default-protocol proxies.
// NOT_FOUND (404) is treated as success — proxy already gone or was never on v2.
func (a *Adapter) Cancel(ctx context.Context, providerOrderID string) error {
	id := firstID(providerOrderID)
	if err := a.client.DeleteProxyV2(ctx, id); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			// Proxy not found on v2 — already deleted or legacy v1 proxy.
			// Treat as success (idempotent).
			return nil
		}
		return mapError(err)
	}
	return nil
}

// ─── Suspend ──────────────────────────────────────────────────────────────────

// Suspend calls POST /api/v2/ipv4/{id}/suspend.
// For default-protocol proxies, suspending the first ID suspends the whole pair.
// Legacy v1 IDs fall back to StopProxy.
func (a *Adapter) Suspend(ctx context.Context, providerOrderID string) error {
	id := firstID(providerOrderID)
	if err := a.client.SuspendProxyV2(ctx, id); err != nil {
		_ = a.client.StopProxy(ctx, id) // v1 fallback
	}
	return nil
}

// ─── Resume ───────────────────────────────────────────────────────────────────

// Resume calls POST /api/v2/ipv4/{id}/resume.
// For default-protocol proxies, resuming the first ID resumes the whole pair.
// Legacy v1 IDs fall back to StartProxy.
func (a *Adapter) Resume(ctx context.Context, providerOrderID string) error {
	id := firstID(providerOrderID)
	if err := a.client.ResumeProxyV2(ctx, id); err != nil {
		_ = a.client.StartProxy(ctx, id) // v1 fallback
	}
	return nil
}

// ─── CheckStatus ──────────────────────────────────────────────────────────────

// CheckStatus calls GET /api/v2/proxies/{id} using the first ID in the pair.
//
// VPM v2 status → Cloudmini:
//
//	"completed"  → "active"
//	"suspended"  → "active"   (proxy is reserved during grace period)
//	"error"      → "failed"
//	others       → "processing"
func (a *Adapter) CheckStatus(ctx context.Context, providerOrderID string) (string, error) {
	id := firstID(providerOrderID)

	// Try v2 first
	proxy, err := a.client.GetProxyV2(ctx, id)
	if err == nil {
		return mapStatusV2(proxy.Status), nil
	}
	// Fallback to v1 for legacy orders
	proxyV1, err2 := a.client.GetProxy(ctx, id)
	if err2 != nil {
		return "", mapError(err)
	}
	return mapStatusV1(proxyV1.Status), nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// splitIDs splits a ProviderOrderID that may contain "|" into individual IDs.
func splitIDs(providerOrderID string) []string {
	parts := strings.Split(providerOrderID, "|")
	var ids []string
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			ids = append(ids, s)
		}
	}
	return ids
}

// firstID returns the first ID from a potentially pipe-separated ProviderOrderID.
func firstID(providerOrderID string) string {
	ids := splitIDs(providerOrderID)
	if len(ids) == 0 {
		return providerOrderID
	}
	return ids[0]
}

// mapStatusV2 converts VPM v2 proxy status strings to Cloudmini constants.
func mapStatusV2(s string) string {
	switch s {
	case "completed", "suspended":
		return "active"
	case "error":
		return "failed"
	default:
		return "processing"
	}
}

// mapStatusV1 converts VPM v1 proxy status strings to Cloudmini constants (legacy).
func mapStatusV1(s string) string {
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
