// Package domain contains core entities, repository interfaces, errors
// for proxy-service.
package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Order status constants.
const (
	OrderPending    = "pending"
	OrderProcessing = "processing"
	OrderActive     = "active"
	OrderExpired    = "expired"
	OrderCancelled  = "cancelled"
	OrderFailed     = "failed"
	OrderRefunded   = "refunded"
)

// Provider entity.
type Provider struct {
	ID          uuid.UUID      `db:"id"`
	Name        string         `db:"name"`
	DisplayName string         `db:"display_name"`
	AdapterType string         `db:"adapter_type"`
	Config      map[string]any `db:"-"` // JSON from db — stored encrypted
	IsActive    bool           `db:"is_active"`
	Priority    int            `db:"priority"`
	CreatedAt   time.Time      `db:"created_at"`
}

// Product entity.
type Product struct {
	ID           uuid.UUID       `db:"id"`
	ProviderID   uuid.UUID       `db:"provider_id"`
	Name         string          `db:"name"`
	ProxyType    string          `db:"proxy_type"`
	Protocol     string          `db:"protocol"`
	Location     string          `db:"location"`
	DurationDays *int            `db:"duration_days"`
	BandwidthGB  *decimal.Decimal `db:"bandwidth_gb"`
	BaseCost     decimal.Decimal `db:"base_cost"`
	IsActive     bool            `db:"is_active"`
	CreatedAt    time.Time       `db:"created_at"`
}

// Order entity.
type Order struct {
	ID               uuid.UUID       `db:"id"`
	OrderNumber      string          `db:"order_number"`
	UserID           uuid.UUID       `db:"user_id"`
	ResellerID       *uuid.UUID      `db:"reseller_id"`
	ProductID        uuid.UUID       `db:"product_id"`
	ProviderID       uuid.UUID       `db:"provider_id"`
	Status           string          `db:"status"`
	Quantity         int             `db:"quantity"`
	UnitPrice        decimal.Decimal `db:"unit_price"`
	TotalAmount      decimal.Decimal `db:"total_amount"`
	ProviderOrderID  string          `db:"provider_order_id"`
	Credentials      string          `db:"credentials"` // AES-256-GCM encrypted JSON
	ActivatedAt      *time.Time      `db:"activated_at"`
	ExpiresAt        *time.Time      `db:"expires_at"`
	CancelledAt      *time.Time      `db:"cancelled_at"`
	CancelReason     string          `db:"cancel_reason"`
	IdempotencyKey   string          `db:"idempotency_key"`
	RequestID        *uuid.UUID      `db:"request_id"`
	CreatedAt        time.Time       `db:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at"`
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
}

// IOrderRepository manages proxy orders.
type IOrderRepository interface {
	Create(ctx context.Context, o *Order) error
	GetByID(ctx context.Context, id uuid.UUID) (*Order, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*Order, error)
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
