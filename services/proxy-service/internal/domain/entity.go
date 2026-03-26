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
	OrderPending    = "pending"     // payment not yet confirmed
	OrderProcessing = "processing"  // paid, waiting for async provider activation
	OrderActive     = "active"      // proxy is live and credentials are available
	OrderExpired    = "expired"     // proxy lifetime ended
	OrderCancelled  = "cancelled"   // cancelled by user or admin
	OrderFailed     = "failed"      // provider purchase failed
	OrderRefunded   = "refunded"    // refunded to wallet
	OrderSuspended  = "suspended"   // admin locked — proxy stopped at provider, user cannot use it
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
	ID               uuid.UUID        `db:"id"               json:"id"`
	OrderNumber      string           `db:"order_number"     json:"order_number"`
	UserID           uuid.UUID        `db:"user_id"          json:"user_id"`
	ResellerID       *uuid.UUID       `db:"reseller_id"      json:"reseller_id,omitempty"`
	ProductID        uuid.UUID        `db:"product_id"       json:"product_id"`
	ProviderID       uuid.UUID        `db:"provider_id"      json:"provider_id"`
	Status           string           `db:"status"           json:"status"`
	Quantity         int              `db:"quantity"         json:"quantity"`
	UnitPrice        decimal.Decimal  `db:"unit_price"       json:"unit_price"`
	TotalAmount      decimal.Decimal  `db:"total_amount"     json:"total_amount"`
	CustomPrice      *decimal.Decimal `db:"custom_price"     json:"custom_price,omitempty"`      // admin override price for renewals
	ProviderOrderID  string           `db:"provider_order_id" json:"provider_order_id,omitempty"`
	Credentials      string           `db:"credentials"       json:"-"` // never expose encrypted creds
	ActivatedAt      *time.Time       `db:"activated_at"     json:"activated_at,omitempty"`
	ExpiresAt        *time.Time       `db:"expires_at"       json:"expires_at,omitempty"`
	CustomExpiresAt  *time.Time       `db:"custom_expires_at" json:"custom_expires_at,omitempty"` // admin override expiry
	CancelledAt      *time.Time       `db:"cancelled_at"     json:"cancelled_at,omitempty"`
	CancelReason     string           `db:"cancel_reason"    json:"cancel_reason,omitempty"`
	AdminNote        string           `db:"admin_note"       json:"admin_note,omitempty"`
	IdempotencyKey   string           `db:"idempotency_key"  json:"-"` // internal
	RequestID        *uuid.UUID       `db:"request_id"       json:"-"` // internal
	CreatedAt        time.Time        `db:"created_at"       json:"created_at"`
	UpdatedAt        time.Time        `db:"updated_at"       json:"updated_at"`
}

// ─── Repository Interfaces ────────────────────────────────────────────────────

// IProviderRepository manages proxy providers.
type IProviderRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Provider, error)
	ListActive(ctx context.Context) ([]*Provider, error)
	ListAll(ctx context.Context) ([]*Provider, error)
	ListByAdapterType(ctx context.Context, adapterType string) ([]*Provider, error)
	Create(ctx context.Context, p *Provider) error
	Update(ctx context.Context, p *Provider) error
	ToggleActive(ctx context.Context, id uuid.UUID, active bool) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// IProductRepository manages proxy products.
type IProductRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Product, error)
	List(ctx context.Context, proxyType, protocol, location string, offset, limit int) ([]*Product, int, error)
	AdminList(ctx context.Context, offset, limit int) ([]*Product, int, error)
	Create(ctx context.Context, p *Product) error
	Update(ctx context.Context, p *Product) error
	Delete(ctx context.Context, id uuid.UUID) error
	ToggleActive(ctx context.Context, id uuid.UUID) error
}

// ─── Order Event ──────────────────────────────────────────────────────────────

// Event type constants for proxy order lifecycle.
const (
	EventOrderCreated   = "order.created"
	EventOrderActivated = "order.activated"
	EventOrderCancelled = "order.cancelled"
	EventOrderPatched   = "order.patched"
	EventOrderFailed    = "order.failed"
	EventOrderExpired   = "order.expired"   // proxy reached expiry, now in grace period
	EventOrderDeleted   = "order.deleted"   // grace period over, proxy permanently deleted
	EventOrderRenewed   = "order.renewed"   // user renewed during grace period
	EventOrderLocked    = "order.locked"    // admin locked (suspended) proxy
	EventOrderUnlocked  = "order.unlocked"  // admin unlocked (resumed) proxy
)

// OrderEvent records an action taken on a proxy order.
type OrderEvent struct {
	ID        uuid.UUID       `db:"id"         json:"id"`
	OrderID   uuid.UUID       `db:"order_id"   json:"order_id"`
	EventType string          `db:"event_type" json:"event_type"`
	Payload   json.RawMessage `db:"payload"    json:"payload,omitempty"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
}

// IOrderEventRepository persist and retrieves order events.
type IOrderEventRepository interface {
	// Log writes an event for the given order. payload may be nil.
	Log(ctx context.Context, orderID uuid.UUID, eventType string, payload map[string]any) error
	// ListByOrder returns all events for an order, oldest first.
	ListByOrder(ctx context.Context, orderID uuid.UUID) ([]*OrderEvent, error)
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
	// UpdateOrder sets custom_price, custom_expires_at and admin_note (user self-service).
	UpdateOrder(ctx context.Context, id uuid.UUID, customPrice *decimal.Decimal, customExpiresAt *time.Time, adminNote string) error
	ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*Order, int, error)
	ListExpiring(ctx context.Context, within time.Duration) ([]*Order, error)
	// ListExpiredActive returns active orders whose effective expiry (custom_expires_at ?? expires_at) has passed.
	// These should be moved to 'expired' status and suspended at the provider.
	ListExpiredActive(ctx context.Context) ([]*Order, error)
	// ListExpiredGrace returns orders in 'expired' status whose effective expiry + grace has passed.
	// These should be permanently cancelled and deleted from the provider.
	ListExpiredGrace(ctx context.Context, grace time.Duration) ([]*Order, error)
}

// IEventPublisher publishes proxy events to NATS.
type IEventPublisher interface {
	PublishOrderCreated(ctx context.Context, order *Order) error
	PublishOrderFulfilled(ctx context.Context, order *Order) error
	PublishOrderCancelled(ctx context.Context, order *Order) error
	PublishOrderFailed(ctx context.Context, orderID uuid.UUID, reason string) error
}
