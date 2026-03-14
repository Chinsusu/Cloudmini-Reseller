// Package domain contains core entities, repository interfaces, errors
// for proxy-service.
package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Order status constants.
const (
	OrderPending     = "pending"     // payment not yet confirmed
	OrderProcessing  = "processing"  // paid, waiting for async provider activation
	OrderActive      = "active"      // proxy is live and credentials are available
	OrderExpired     = "expired"     // proxy lifetime ended
	OrderCancelled   = "cancelled"   // cancelled by user or admin
	OrderFailed      = "failed"      // provider purchase failed
	OrderRefunded    = "refunded"    // refunded to wallet
)

// Provider entity.
type Provider struct {
	ID          uuid.UUID       `db:"id"          json:"id"`
	Name        string          `db:"name"         json:"name"`
	DisplayName string          `db:"display_name" json:"display_name"`
	AdapterType string          `db:"adapter_type" json:"adapter_type"`
	Config      json.RawMessage `db:"config"       json:"-"` // JSONB — never expose config to frontend
	IsActive    bool            `db:"is_active"    json:"is_active"`
	Priority    int             `db:"priority"     json:"priority"`
	CreatedAt   time.Time       `db:"created_at"   json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at"   json:"updated_at"`
}

// Product entity.
type Product struct {
	ID           uuid.UUID        `db:"id"            json:"id"`
	ProviderID   uuid.UUID        `db:"provider_id"   json:"provider_id"`
	Name         string           `db:"name"          json:"name"`
	ProxyType    string           `db:"proxy_type"    json:"proxy_type"`
	Protocol     string           `db:"protocol"      json:"protocol"`
	Location     string           `db:"location"      json:"location"`
	DurationDays *int             `db:"duration_days" json:"duration_days"`
	BandwidthGB  *decimal.Decimal `db:"bandwidth_gb"  json:"bandwidth_gb"`
	BaseCost     decimal.Decimal  `db:"base_cost"     json:"base_cost"`
	IsActive     bool             `db:"is_active"     json:"is_active"`
	Metadata     json.RawMessage  `db:"metadata"      json:"metadata,omitempty"`
	CreatedAt    time.Time        `db:"created_at"    json:"created_at"`
}

// Order entity.
type Order struct {
	ID               uuid.UUID       `db:"id"               json:"id"`
	OrderNumber      string          `db:"order_number"     json:"order_number"`
	UserID           uuid.UUID       `db:"user_id"          json:"user_id"`
	ResellerID       *uuid.UUID      `db:"reseller_id"      json:"reseller_id,omitempty"`
	ProductID        uuid.UUID       `db:"product_id"       json:"product_id"`
	ProviderID       uuid.UUID       `db:"provider_id"      json:"provider_id"`
	Status           string          `db:"status"           json:"status"`
	Quantity         int             `db:"quantity"         json:"quantity"`
	UnitPrice        decimal.Decimal `db:"unit_price"       json:"unit_price"`
	TotalAmount      decimal.Decimal `db:"total_amount"     json:"total_amount"`
	ProviderOrderID  string          `db:"provider_order_id" json:"provider_order_id,omitempty"`
	Credentials      string          `db:"credentials"       json:"-"` // never expose encrypted creds
	ActivatedAt      *time.Time      `db:"activated_at"     json:"activated_at,omitempty"`
	ExpiresAt        *time.Time      `db:"expires_at"       json:"expires_at,omitempty"`
	CancelledAt      *time.Time      `db:"cancelled_at"     json:"cancelled_at,omitempty"`
	CancelReason     string          `db:"cancel_reason"    json:"cancel_reason,omitempty"`
	IdempotencyKey   string          `db:"idempotency_key"  json:"-"` // internal
	RequestID        *uuid.UUID      `db:"request_id"       json:"-"` // internal
	CreatedAt        time.Time       `db:"created_at"       json:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at"       json:"updated_at"`
}

// ─── Repository Interfaces ────────────────────────────────────────────────────

// IProviderRepository manages proxy providers.
type IProviderRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Provider, error)
	ListActive(ctx context.Context) ([]*Provider, error)
}

// IProductRepository manages proxy products.
type IProductRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Product, error)
	List(ctx context.Context, proxyType, protocol, location string, offset, limit int) ([]*Product, int, error)
	AdminList(ctx context.Context, offset, limit int) ([]*Product, int, error) // all products, no is_active filter
	Create(ctx context.Context, p *Product) error
	Update(ctx context.Context, p *Product) error
	Delete(ctx context.Context, id uuid.UUID) error
	ToggleActive(ctx context.Context, id uuid.UUID) error
}

// IOrderRepository manages proxy orders.
type IOrderRepository interface {
	Create(ctx context.Context, o *Order) error
	GetByID(ctx context.Context, id uuid.UUID) (*Order, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*Order, error)
	// GetByProviderOrderID finds an order using the provider's order reference ID.
	// Used by webhook handlers to locate the Cloudmini order after async activation.
	GetByProviderOrderID(ctx context.Context, providerOrderID string) (*Order, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdateAfterPurchase(ctx context.Context, id uuid.UUID, providerOrderID, credentials string, activatedAt, expiresAt *time.Time) error
	ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*Order, int, error)
	ListExpiring(ctx context.Context, within time.Duration) ([]*Order, error)
}

// IEventPublisher publishes proxy events to NATS.
type IEventPublisher interface {
	PublishOrderCreated(ctx context.Context, order *Order) error
	PublishOrderFulfilled(ctx context.Context, order *Order) error
	PublishOrderCancelled(ctx context.Context, order *Order) error
	PublishOrderFailed(ctx context.Context, orderID uuid.UUID, reason string) error
}
