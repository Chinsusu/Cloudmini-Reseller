package cpm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/pvp/proxy-service/internal/providers"
)

// ProviderName is the adapter_type registered in the proxy.providers table.
const ProviderName = "cpm"

// Config holds CPM connection settings.
// Stored encrypted in proxy.providers.config JSONB.
type Config struct {
	BaseURL string `json:"base_url"` // e.g. "https://cz.resvn.net"
	APIKey  string `json:"api_key"`  // API key (X-API-Key header)
}

// Adapter implements providers.IProxyProvider for the CPM IPv4-Res API V1.
// CPM V1 is async: POST /api/v1/ipv4-res returns status="creating",
// we poll until "running" before returning credentials.
//
// Protocol behaviour:
//   - "default" → response has both port_http and port_socks → 2 credentials (HTTP + SOCKS5)
//   - "http"    → port_http only → 1 credential
//   - "socks5"  → port_socks only → 1 credential
//   - vmess/vless/shadowsocks/trojan → use connect_url or connection_string
//
// ProviderOrderID = plain proxy UUID (one DELETE removes all ports).
type Adapter struct {
	client *Client
	cfg    Config
	logger *slog.Logger
}

// NewAdapter creates a new CPM adapter.
func NewAdapter(cfg Config) *Adapter {
	return &Adapter{
		client: NewClient(cfg.BaseURL, cfg.APIKey),
		cfg:    cfg,
		logger: slog.Default(),
	}
}

// NewAdapterWithClient creates an adapter with an injected client (for testing).
func NewAdapterWithClient(cfg Config, client *Client) *Adapter {
	return &Adapter{client: client, cfg: cfg, logger: slog.Default()}
}

// ─── Purchase ─────────────────────────────────────────────────────────────────

// Purchase calls POST /api/v1/proxies, polls until running, and returns credentials.
//
// PurchaseRequest.Metadata keys:
//   - "protocol"           — "default"|"http"|"socks5"|"vmess"|"vless"|"shadowsocks"|"trojan"
//   - "group_id"           — CPM region/group UUID
//   - "listen_ip"          — specific IP (optional; CPM auto-selects if empty)
//   - "ip_address_id"      — specific IP UUID
//   - "ip_range_id"        — specific IP range UUID
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

	createReq := CreateProxyRequest{
		Protocol:    protocol,
		GroupID:     m["group_id"],
		ListenIP:    m["listen_ip"],
		IPAddressID: m["ip_address_id"],
		IPRangeID:   m["ip_range_id"],
	}
	// Backward compat: old metadata used "ipv4" key → map to listen_ip
	if createReq.ListenIP == "" && m["ipv4"] != "" {
		createReq.ListenIP = m["ipv4"]
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

	a.logger.InfoContext(ctx, "cpm.Purchase",
		slog.String("protocol", protocol),
		slog.String("listen_ip", createReq.ListenIP),
		slog.String("group_id", createReq.GroupID),
		slog.String("req_protocol", req.Protocol),
		slog.String("meta_protocol", m["protocol"]),
	)

	// Create proxy and wait until running (async V1 flow)
	proxy, err := a.client.CreateProxyAndWait(ctx, createReq)
	if err != nil {
		return nil, mapError(err)
	}

	// Check for error status after polling
	if proxy.Status == "error" {
		return nil, fmt.Errorf("%w: cpm proxy creation failed with status 'error'", providers.ErrProviderUnavailable)
	}

	// Build credentials from response
	creds := buildCredentials(proxy)
	if len(creds) == 0 {
		return nil, fmt.Errorf("%w: cpm returned no usable credentials", providers.ErrProviderUnavailable)
	}

	return &providers.PurchaseResult{
		ProviderOrderID: proxy.ID,
		Credentials:     creds,
	}, nil
}

// buildCredentials creates ProxyCredential slice from a ProxySummary.
func buildCredentials(p *ProxySummary) []providers.ProxyCredential {
	host := p.effectiveHost()
	var creds []providers.ProxyCredential

	switch p.Protocol {
	case "default":
		// Dual port: create both HTTP and SOCKS5 credentials
		if p.PortHTTP > 0 {
			creds = append(creds, providers.ProxyCredential{
				Host: host, Port: p.PortHTTP,
				Username: p.AuthUser, Password: p.AuthPass,
				Protocol: "http",
			})
		}
		if p.PortSOCKS > 0 {
			creds = append(creds, providers.ProxyCredential{
				Host: host, Port: p.PortSOCKS,
				Username: p.AuthUser, Password: p.AuthPass,
				Protocol: "socks5",
			})
		}
	case "http":
		port := p.PortHTTP
		if port == 0 {
			port = p.Port
		}
		if port > 0 {
			creds = append(creds, providers.ProxyCredential{
				Host: host, Port: port,
				Username: p.AuthUser, Password: p.AuthPass,
				Protocol: "http",
			})
		}
	case "socks5":
		port := p.PortSOCKS
		if port == 0 {
			port = p.Port
		}
		if port > 0 {
			creds = append(creds, providers.ProxyCredential{
				Host: host, Port: port,
				Username: p.AuthUser, Password: p.AuthPass,
				Protocol: "socks5",
			})
		}
	default:
		// vmess, vless, shadowsocks, trojan — use connect_url or connection_string
		connStr := p.ConnectURL
		if connStr == "" {
			connStr = p.ConnectionString
		}
		creds = append(creds, providers.ProxyCredential{
			Host: host, Port: p.effectivePort(),
			Username:         p.AuthUser,
			Password:         p.AuthPass,
			Protocol:         p.Protocol,
			ConnectionString: connStr,
		})
	}
	return creds
}

// ─── Cancel ───────────────────────────────────────────────────────────────────

// Cancel permanently deletes a proxy from CPM.
// DELETE /api/v1/ipv4-res/{id} → 204 No Content
// NOT_FOUND (404) is treated as success (idempotent).
func (a *Adapter) Cancel(ctx context.Context, providerOrderID string) error {
	if err := a.client.DeleteProxy(ctx, providerOrderID); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return nil // already deleted
		}
		return mapError(err)
	}
	return nil
}

// ─── Suspend ──────────────────────────────────────────────────────────────────

// Suspend stops a running proxy (V1 API uses stop instead of lock).
// POST /api/v1/ipv4-res/{id}/stop
func (a *Adapter) Suspend(ctx context.Context, providerOrderID string) error {
	if err := a.client.StopProxy(ctx, providerOrderID); err != nil {
		return mapError(err)
	}
	return nil
}

// ─── Resume ───────────────────────────────────────────────────────────────────

// Resume starts a stopped proxy.
// POST /api/v1/ipv4-res/{id}/start
func (a *Adapter) Resume(ctx context.Context, providerOrderID string) error {
	if err := a.client.StartProxy(ctx, providerOrderID); err != nil {
		return mapError(err)
	}
	return nil
}

// ─── CheckStatus ──────────────────────────────────────────────────────────────

// CheckStatus calls GET /api/v1/ipv4-res/{id}.
//
// CPM status → Cloudmini:
//
//	"running"   → "active"
//	"stopped"   → "active"   (proxy is reserved, just paused)
//	"creating"  → "processing"
//	"error"     → "failed"
func (a *Adapter) CheckStatus(ctx context.Context, providerOrderID string) (string, error) {
	proxy, err := a.client.GetProxy(ctx, providerOrderID)
	if err != nil {
		return "", mapError(err)
	}
	return mapStatus(proxy.Status), nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// mapStatus converts CPM proxy status strings to Cloudmini constants.
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

// mapError converts CPM APIError to providers error types.
func mapError(err error) error {
	apiErr, ok := err.(*APIError)
	if !ok {
		return err
	}
	switch apiErr.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("%w: cpm auth failed: %s", providers.ErrProviderUnavailable, apiErr.Code)
	case http.StatusUnprocessableEntity, http.StatusBadRequest:
		return fmt.Errorf("%w: %s", providers.ErrInvalidConfig, apiErr.Message)
	case http.StatusNotFound:
		return fmt.Errorf("%w: proxy not found in cpm", providers.ErrProviderUnavailable)
	default:
		return fmt.Errorf("%w: %s", providers.ErrProviderUnavailable, apiErr.Message)
	}
}
