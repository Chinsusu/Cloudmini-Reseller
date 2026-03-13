package proxy_cheap

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/pvp/proxy-service/internal/domain"
)

// WehookFulfiller is the interface the webhook handler needs to fulfill an order.
type WebhookFulfiller interface {
	// FulfillFromProxyCheap is called when a proxy.status.changed → ACTIVE webhook is received.
	// It fetches the proxy credentials, encrypts them, and marks the order active.
	FulfillFromProxyCheap(ctx context.Context, providerOrderID string, proxy *Proxy) error
}

// WebhookHandler handles inbound webhook events from Proxy-Cheap.
type WebhookHandler struct {
	client    *Client
	secret    string
	fulfiller WebhookFulfiller
	logger    *slog.Logger
}

// NewWebhookHandler creates a WebhookHandler.
func NewWebhookHandler(client *Client, secret string, fulfiller WebhookFulfiller, logger *slog.Logger) *WebhookHandler {
	return &WebhookHandler{
		client:    client,
		secret:    secret,
		fulfiller: fulfiller,
		logger:    logger,
	}
}

// HandleEvent is the HTTP handler for POST /webhooks/proxy-cheap.
// It reads the body, verifies the HMAC signature, then routes by event type.
func (h *WebhookHandler) HandleEvent(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	eventName := r.Header.Get("Webhook-Event")
	eventID := r.Header.Get("Webhook-Id")
	sig := r.Header.Get("Webhook-Signature")

	if !h.verifySignature(sig, eventName, eventID, body) {
		h.logger.WarnContext(r.Context(), "proxy-cheap webhook: invalid HMAC signature",
			slog.String("event", eventName),
			slog.String("event_id", eventID),
		)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	h.logger.InfoContext(r.Context(), "proxy-cheap webhook received",
		slog.String("event", eventName),
		slog.String("event_id", eventID),
	)

	switch eventName {
	case WebhookEventStatusChanged:
		h.handleStatusChanged(r.Context(), body)
	case WebhookEventBandwidthAdded:
		h.handleBandwidthAdded(r.Context(), body)
	case WebhookEventMaintenanceWindowCreated:
		h.handleMaintenanceWindowCreated(r.Context(), body)
	case WebhookEventMaintenanceWindowCancelled:
		h.handleMaintenanceWindowCancelled(r.Context(), body)
	default:
		h.logger.WarnContext(r.Context(), "proxy-cheap webhook: unknown event",
			slog.String("event", eventName),
		)
	}

	// Always return 200 — Proxy-Cheap does not retry
	w.WriteHeader(http.StatusOK)
}

// verifySignature validates the HMAC-SHA256 webhook signature.
// Input to HMAC: "sha256" + eventName + eventID + body (minified JSON) + secret
func (h *WebhookHandler) verifySignature(sig, eventName, eventID string, body []byte) bool {
	if !strings.HasPrefix(sig, "sha256=") {
		return false
	}
	input := "sha256" + eventName + eventID + string(body) + h.secret
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write([]byte(input))
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expected))
}

// handleStatusChanged processes proxy.status.changed events.
// When status becomes ACTIVE, it fetches proxy details and fulfills the Cloudmini order.
func (h *WebhookHandler) handleStatusChanged(ctx context.Context, body []byte) {
	var payload WebhookStatusChanged
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.ErrorContext(ctx, "proxy-cheap webhook: decode status_changed", slog.String("error", err.Error()))
		return
	}

	h.logger.InfoContext(ctx, "proxy-cheap status changed",
		slog.Int64("proxy_id", payload.ProxyID),
		slog.String("old_status", payload.OldStatus),
		slog.String("new_status", payload.Status),
	)

	if payload.Status != ProxyStatusActive {
		// Nothing to do for PENDING/INITIATING/EXPIRED/CANCELED here
		// Expiry is handled by the cron job sync-provider-status
		return
	}

	// Fetch full proxy details to get connection info + credentials
	proxy, err := h.client.GetProxy(ctx, payload.ProxyID)
	if err != nil {
		h.logger.ErrorContext(ctx, "proxy-cheap webhook: fetch proxy details",
			slog.Int64("proxy_id", payload.ProxyID),
			slog.String("error", err.Error()),
		)
		return
	}

	// We need the providerOrderID to match the Cloudmini order.
	// The adapter stores providerOrderID = order UUID from Proxy-Cheap execute response.
	// Since the webhook only gives proxyId (int64), we look up using the proxy's order
	// by querying GET /orders/:id/proxies. However we only have proxyId here.
	// Strategy: store proxyId→orderID mapping OR query by proxyId in the usecase.
	// For simplicity, we pass the proxy directly to FulfillFromProxyCheap which will
	// use proxyId to find the associated Cloudmini order via repository.
	if err := h.fulfiller.FulfillFromProxyCheap(ctx, fmt.Sprintf("%d", payload.ProxyID), proxy); err != nil {
		if err == domain.ErrOrderNotFound {
			// Could be from another system or already fulfilled — ignore
			h.logger.WarnContext(ctx, "proxy-cheap webhook: order not found for proxy",
				slog.Int64("proxy_id", payload.ProxyID),
			)
			return
		}
		h.logger.ErrorContext(ctx, "proxy-cheap webhook: fulfill order",
			slog.Int64("proxy_id", payload.ProxyID),
			slog.String("error", err.Error()),
		)
	}
}

// handleBandwidthAdded processes proxy.bandwidth.added events (Phase 1: log only).
func (h *WebhookHandler) handleBandwidthAdded(ctx context.Context, body []byte) {
	var payload WebhookBandwidthAdded
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.ErrorContext(ctx, "proxy-cheap webhook: decode bandwidth_added", slog.String("error", err.Error()))
		return
	}
	h.logger.InfoContext(ctx, "proxy-cheap bandwidth added",
		slog.Int64("proxy_id", payload.ProxyID),
		slog.Int("traffic_gb", payload.TrafficInGB),
	)
}

// handleMaintenanceWindowCreated processes maintenance window start events (Phase 1: log only).
func (h *WebhookHandler) handleMaintenanceWindowCreated(ctx context.Context, body []byte) {
	var payload WebhookMaintenanceWindow
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.ErrorContext(ctx, "proxy-cheap webhook: decode maintenance_created", slog.String("error", err.Error()))
		return
	}
	h.logger.WarnContext(ctx, "proxy-cheap maintenance window started",
		slog.Int64("proxy_id", payload.ProxyID),
		slog.String("window_id", payload.MaintenanceWindowID),
	)
}

// handleMaintenanceWindowCancelled processes maintenance window end events (Phase 1: log only).
func (h *WebhookHandler) handleMaintenanceWindowCancelled(ctx context.Context, body []byte) {
	var payload WebhookMaintenanceWindow
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.ErrorContext(ctx, "proxy-cheap webhook: decode maintenance_cancelled", slog.String("error", err.Error()))
		return
	}
	h.logger.InfoContext(ctx, "proxy-cheap maintenance window cancelled (server back online)",
		slog.Int64("proxy_id", payload.ProxyID),
		slog.String("window_id", payload.MaintenanceWindowID),
	)
}
