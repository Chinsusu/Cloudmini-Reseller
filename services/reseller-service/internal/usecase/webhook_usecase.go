package usecase

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pvp/reseller-service/internal/domain"
)

// WebhookUsecase manages reseller webhooks and outgoing delivery.
type WebhookUsecase struct {
	webhookRepo domain.IWebhookRepository
	logger      *slog.Logger
}

func NewWebhookUsecase(webhookRepo domain.IWebhookRepository, logger *slog.Logger) *WebhookUsecase {
	return &WebhookUsecase{webhookRepo: webhookRepo, logger: logger}
}

// CreateWebhook registers a reseller webhook endpoint.
func (u *WebhookUsecase) CreateWebhook(ctx context.Context, resellerID uuid.UUID, webhookURL, secret string, events []string) (*domain.ResellerWebhook, error) {
	w := &domain.ResellerWebhook{
		ID:         uuid.New(),
		ResellerID: resellerID,
		URL:        webhookURL,
		Secret:     secret,
		Events:     events,
		IsActive:   true,
		CreatedAt:  time.Now(),
	}
	if err := u.webhookRepo.Create(ctx, w); err != nil {
		return nil, fmt.Errorf("CreateWebhook: %w", err)
	}
	return w, nil
}

// ListWebhooks returns all webhooks for a reseller.
func (u *WebhookUsecase) ListWebhooks(ctx context.Context, resellerID uuid.UUID) ([]*domain.ResellerWebhook, error) {
	return u.webhookRepo.ListByReseller(ctx, resellerID)
}

// DeleteWebhook removes a webhook.
func (u *WebhookUsecase) DeleteWebhook(ctx context.Context, id, resellerID uuid.UUID) error {
	w, err := u.webhookRepo.GetByID(ctx, id)
	if err != nil {
		return domain.ErrWebhookNotFound
	}
	if w.ResellerID != resellerID {
		return domain.ErrForbidden
	}
	return u.webhookRepo.Delete(ctx, id)
}

// DeliverToReseller sends an event to all active webhooks for a reseller.
// Called when platform events occur (order fulfilled, billing charged, etc.)
// Uses HMAC-SHA256 signing: X-PVP-Signature: sha256=<hex>
// Retries: 0s, 30s, 5m (3 attempts total, best-effort, background goroutine).
func (u *WebhookUsecase) DeliverToReseller(ctx context.Context, resellerID uuid.UUID, eventType string, payload any) {
	webhooks, err := u.webhookRepo.ListByReseller(ctx, resellerID)
	if err != nil || len(webhooks) == 0 {
		return
	}

	data, err := json.Marshal(map[string]any{
		"event":     eventType,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data":      payload,
	})
	if err != nil {
		return
	}

	for _, w := range webhooks {
		if !w.IsActive {
			continue
		}
		// Check if this webhook subscribes to this event
		subscribed := false
		for _, e := range w.Events {
			if e == "*" || e == eventType {
				subscribed = true
				break
			}
		}
		if !subscribed {
			continue
		}

		wCopy := *w
		go u.deliverWithRetry(wCopy.URL, wCopy.Secret, data, 3)
	}
}

func (u *WebhookUsecase) deliverWithRetry(url, secret string, data []byte, maxRetries int) {
	delays := []time.Duration{0, 30 * time.Second, 5 * time.Minute}
	for i := 0; i < maxRetries; i++ {
		if delays[i] > 0 {
			time.Sleep(delays[i])
		}
		if err := u.deliver(url, secret, data); err == nil {
			return
		}
		u.logger.Warn("webhook delivery failed",
			slog.String("url", url),
			slog.Int("attempt", i+1),
		)
	}
}

func (u *WebhookUsecase) deliver(url, secret string, data []byte) error {
	sig := computeHMAC(secret, data)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("webhook: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PVP-Signature", "sha256="+sig)
	req.Header.Set("X-PVP-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))
	req.Header.Set("X-PVP-Delivery-ID", uuid.New().String())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook: do: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook: status %d", resp.StatusCode)
	}
	return nil
}

func computeHMAC(secret string, data []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
