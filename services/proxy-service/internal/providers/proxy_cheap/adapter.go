package proxy_cheap

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pvp/proxy-service/internal/providers"
)

// ProviderName is the adapter name registered in the provider registry.
const ProviderName = "proxy_cheap"

// Config holds Proxy-Cheap credentials and settings.
// Retrieved from Provider.Config (decrypted JSONB).
type Config struct {
	APIKey         string `json:"api_key"`
	APISecret      string `json:"api_secret"`
	WebhookSecret  string `json:"webhook_secret"`
	// ServiceID and PlanID encode the product type for this adapter.
	// These are set per Product row in the DB via Provider.Config defaults.
	// PurchaseRequest.ProductID is a composite key: "<serviceId>|<planId>|<country>|<ispId>"
}

// Adapter implements providers.IProxyProvider for Proxy-Cheap.
type Adapter struct {
	client  *Client
	cfg     Config
}

// NewAdapter creates a new Proxy-Cheap adapter.
func NewAdapter(cfg Config) *Adapter {
	client := NewClient(cfg.APIKey, cfg.APISecret)
	return &Adapter{client: client, cfg: cfg}
}

// NewAdapterWithClient creates an adapter with an injected client (for testing).
func NewAdapterWithClient(cfg Config, client *Client) *Adapter {
	return &Adapter{client: client, cfg: cfg}
}

// Purchase calls POST /v2/order/:serviceId/execute.
// Since Proxy-Cheap activates proxies asynchronously (via webhook), this returns
// a PurchaseResult with empty Credentials. The webhook handler will later call
// WebhookUsecase.FulfillOrder() when the proxy becomes ACTIVE.
func (a *Adapter) Purchase(ctx context.Context, req providers.PurchaseRequest) (*providers.PurchaseResult, error) {
	// ProductID encodes the order config as JSON in Metadata
	// Expected keys: serviceId, planId, country, ispId, periodMonths, traffic, packageId
	serviceID, execReq, err := buildExecuteRequest(req)
	if err != nil {
		return nil, fmt.Errorf("proxy_cheap.Purchase: build request: %w", err)
	}

	resp, err := a.client.Execute(ctx, serviceID, execReq)
	if err != nil {
		return nil, mapProviderError(err)
	}

	return &providers.PurchaseResult{
		ProviderOrderID: resp.ID,
		Credentials:     nil, // populated async via webhook
	}, nil
}

// Cancel is a no-op: Proxy-Cheap has no cancel API endpoint for running proxies.
// The order will expire naturally or the user can request a refund from support.
func (a *Adapter) Cancel(_ context.Context, _ string) error {
	return nil
}

// CheckStatus polls GET /orders/:id/proxies to get the current status of the first proxy.
// Returns Cloudmini status constants: "processing", "active", "expired", "cancelled".
func (a *Adapter) CheckStatus(ctx context.Context, providerOrderID string) (string, error) {
	proxies, err := a.client.GetOrderProxies(ctx, providerOrderID)
	if err != nil {
		return "", fmt.Errorf("proxy_cheap.CheckStatus: %w", err)
	}
	if len(proxies) == 0 {
		return "processing", nil
	}
	return mapStatus(proxies[0].Status), nil
}

// Suspend is a no-op for Proxy-Cheap — the provider does not expose a suspend API.
// Proxy-Cheap orders expire naturally; grace-period tracking is done in Cloudmini only.
func (a *Adapter) Suspend(_ context.Context, _ string) error { return nil }

// ─── Helpers ─────────────────────────────────────────────────────────────────

// buildExecuteRequest converts a PurchaseRequest to a Proxy-Cheap ExecuteRequest.
// PurchaseRequest.Metadata keys:
//   - "service_id"     (required) e.g. "static-residential-ipv4"
//   - "plan_id"        (optional) e.g. "basic"
//   - "package_id"     (optional) for datacenter-ipv6
//   - "country"        (optional) ISO country code
//   - "isp_id"         (optional) ISP UUID
//   - "period_months"  (optional, default "1")
//   - "traffic_gb"     (optional) for rotating proxies
func buildExecuteRequest(req providers.PurchaseRequest) (string, ExecuteRequest, error) {
	m := req.Metadata
	if m == nil {
		m = map[string]string{}
	}

	serviceID := m["service_id"]
	if serviceID == "" {
		return "", ExecuteRequest{}, fmt.Errorf("metadata.service_id is required")
	}

	execReq := ExecuteRequest{
		PlanID:     m["plan_id"],
		PackageID:  m["package_id"],
		Quantity:   req.Quantity,
		Country:    req.Country,
		ISPID:      m["isp_id"],
		CouponCode: m["coupon_code"],
	}

	// Period
	periodMonths := 1
	if pm := m["period_months"]; pm != "" {
		_, err := fmt.Sscanf(pm, "%d", &periodMonths)
		if err != nil {
			periodMonths = 1
		}
	}
	if periodMonths > 0 {
		execReq.Period = &PeriodSpec{Unit: "months", Value: periodMonths}
	}

	// Traffic (GB) for rotating proxies
	if tg := m["traffic_gb"]; tg != "" {
		var traffic int
		_, _ = fmt.Sscanf(tg, "%d", &traffic)
		execReq.Traffic = traffic
	}

	// Auto-extend
	execReq.AutoExtend = &AutoExtendSpec{IsEnabled: true}

	return serviceID, execReq, nil
}

// mapStatus converts Proxy-Cheap status strings to Cloudmini order status constants.
func mapStatus(s string) string {
	switch s {
	case ProxyStatusActive:
		return "active"
	case ProxyStatusExpired:
		return "expired"
	case ProxyStatusCanceled:
		return "cancelled"
	default: // PENDING, INITIATING
		return "processing"
	}
}

// mapProviderError converts Proxy-Cheap API errors to internal error types.
func mapProviderError(err error) error {
	apiErr, ok := err.(*APIError)
	if !ok {
		return err
	}
	switch apiErr.StatusCode {
	case http.StatusUnprocessableEntity:
		return fmt.Errorf("%w: %s", providers.ErrInvalidConfig, apiErr.Body)
	case http.StatusForbidden:
		return fmt.Errorf("%w: proxy-cheap auth failed", providers.ErrProviderUnavailable)
	case http.StatusPaymentRequired:
		return fmt.Errorf("%w: proxy-cheap account balance insufficient", providers.ErrProviderBalance)
	default:
		return fmt.Errorf("%w: %s", providers.ErrProviderUnavailable, apiErr.Body)
	}
}
