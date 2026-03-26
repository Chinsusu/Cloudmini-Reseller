// Package usecase — webhook_usecase.go handles async proxy fulfillment from
// Proxy-Cheap webhooks. When a proxy.status.changed (→ ACTIVE) event arrives,
// this usecase locates the pending Cloudmini order, encrypts the proxy credentials,
// marks the order active, and confirms the billing hold.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	cryptopkg "github.com/pvp/pkg/crypto"
	"github.com/pvp/proxy-service/internal/domain"
	"github.com/pvp/proxy-service/internal/providers"
	proxycheap "github.com/pvp/proxy-service/internal/providers/proxy_cheap"
)

// WebhookUsecase handles async provider events (webhook-triggered fulfillment).
type WebhookUsecase struct {
	orderRepo     domain.IOrderRepository
	productRepo   domain.IProductRepository
	billingClient BillingClient
	cipher        *cryptopkg.Cipher
	eventPub      domain.IEventPublisher
	orderEvtRepo  domain.IOrderEventRepository
	logger        *slog.Logger
}

// NewWebhookUsecase creates a WebhookUsecase.
func NewWebhookUsecase(
	orderRepo domain.IOrderRepository,
	productRepo domain.IProductRepository,
	billingClient BillingClient,
	cipher *cryptopkg.Cipher,
	eventPub domain.IEventPublisher,
	orderEvtRepo domain.IOrderEventRepository,
	logger *slog.Logger,
) *WebhookUsecase {
	return &WebhookUsecase{
		orderRepo:     orderRepo,
		productRepo:   productRepo,
		billingClient: billingClient,
		cipher:        cipher,
		eventPub:      eventPub,
		orderEvtRepo:  orderEvtRepo,
		logger:        logger,
	}
}

// FulfillFromProxyCheap is called when a proxy.status.changed → ACTIVE webhook arrives.
// It finds the Cloudmini order by providerOrderID, encrypts the proxy credentials,
// marks the order active, confirms the billing hold, and publishes the fulfilled event.
// Implements proxy_cheap.WebhookFulfiller interface.
func (u *WebhookUsecase) FulfillFromProxyCheap(ctx context.Context, providerOrderID string, proxy *proxycheap.Proxy) error {
	// Find the Cloudmini order by providerOrderID
	order, err := u.orderRepo.GetByProviderOrderID(ctx, providerOrderID)
	if err != nil {
		return fmt.Errorf("FulfillFromProxyCheap: look up order: %w", err)
	}

	// Only fulfill orders that are in processing state
	if order.Status != domain.OrderProcessing {
		u.logger.InfoContext(ctx, "FulfillFromProxyCheap: order not in processing state, skipping",
			slog.String("order_id", order.ID.String()),
			slog.String("status", order.Status),
		)
		return nil
	}

	// Map Proxy-Cheap proxy to Cloudmini ProxyCredential
	conn := proxy.Connection
	protocol := "http"
	port := conn.HTTPPort
	if proxy.ProxyType == "SOCKS5" {
		protocol = "socks5"
		port = conn.SOCKS5Port
	}

	creds := []providers.ProxyCredential{
		{
			Host:     conn.ConnectIP,
			Port:     port,
			Username: proxy.Authentication.Username,
			Password: proxy.Authentication.Password,
			Protocol: protocol,
			Country:  proxy.CountryCode,
		},
	}

	credJSON, _ := json.Marshal(creds)
	encryptedCreds, err := u.cipher.Encrypt(credJSON)
	if err != nil {
		return fmt.Errorf("FulfillFromProxyCheap: encrypt credentials: %w", err)
	}

	// Determine expiry from proxy
	activatedAt := time.Now()
	expiresAt := proxy.ExpiresAt
	var expiresAtPtr *time.Time
	if !expiresAt.IsZero() {
		expiresAtPtr = &expiresAt
	} else {
		// Fall back to product duration
		product, err := u.productRepo.GetByID(ctx, *order.ProductID)
		if err == nil && product.DurationDays != nil {
			t := activatedAt.AddDate(0, 0, *product.DurationDays)
			expiresAtPtr = &t
		}
	}

	// Update order: credentials + status=active + timestamps
	if err := u.orderRepo.UpdateAfterPurchase(ctx,
		order.ID,
		providerOrderID,
		encryptedCreds,
		&activatedAt,
		expiresAtPtr,
	); err != nil {
		return fmt.Errorf("FulfillFromProxyCheap: update order: %w", err)
	}

	// Confirm the billing hold (deduct funds from wallet)
	if err := u.billingClient.ConfirmHold(ctx, order.UserID, order.TotalAmount, "proxy_order", order.ID, fmt.Sprintf("Proxy %s", order.OrderNumber)); err != nil {
		// Log but don't fail — order is already activated; billing reconciliation can handle
		u.logger.ErrorContext(ctx, "FulfillFromProxyCheap: confirm billing hold",
			slog.String("order_id", order.ID.String()),
			slog.String("error", err.Error()),
		)
	}

	// Publish order fulfilled event for notifications
	order.Status = domain.OrderActive
	order.ActivatedAt = &activatedAt
	order.ExpiresAt = expiresAtPtr
	order.Credentials = encryptedCreds

	// Log order.activated event (async path via webhook)
	_ = u.orderEvtRepo.Log(ctx, order.ID, domain.EventOrderActivated, map[string]any{
		"provider_order_id": providerOrderID,
		"amount":            order.TotalAmount.String(),
	})

	go func() { _ = u.eventPub.PublishOrderFulfilled(context.Background(), order) }()

	u.logger.InfoContext(ctx, "proxy order fulfilled via webhook",
		slog.String("order_id", order.ID.String()),
		slog.String("provider_order_id", providerOrderID),
	)
	return nil
}
