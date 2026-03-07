// Package domain contains core entities, repository interfaces, and errors
// for reseller-service.
package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Reseller account status.
const (
	StatusPending   = "pending"
	StatusApproved  = "approved"
	StatusSuspended = "suspended"
)

// ResellerAccount represents a platform reseller.
type ResellerAccount struct {
	ID            uuid.UUID       `db:"id"`
	UserID        uuid.UUID       `db:"user_id"` // linked user account
	CompanyName   string          `db:"company_name"`
	Slug          *string         `db:"slug"`           // for white-label subdomain (nullable)
	Email         string          `db:"email"`
	Phone         string          `db:"phone"`
	Address       string          `db:"address"`
	TaxID         string          `db:"tax_id"`
	Status        string          `db:"status"` // pending|approved|suspended
	APIKeyPrefix  string          `db:"api_key_prefix"`
	WalletID      *uuid.UUID      `db:"wallet_id"`
	CreditLimit   decimal.Decimal `db:"credit_limit"`
	CommissionPct decimal.Decimal `db:"commission_pct"` // 0-100
	Notes         string          `db:"notes"`
	ApprovedAt    *time.Time      `db:"approved_at"`
	SuspendedAt   *time.Time      `db:"suspended_at"`
	SuspendReason string          `db:"suspend_reason"`
	CreatedAt     time.Time       `db:"created_at"`
	UpdatedAt     time.Time       `db:"updated_at"`
}

// PricingOverride holds a reseller's custom sell price for a product.
type PricingOverride struct {
	ID          uuid.UUID       `db:"id"`
	ResellerID  uuid.UUID       `db:"reseller_id"`
	ProductID   uuid.UUID       `db:"product_id"`
	ProductType string          `db:"product_type"` // proxy|vps
	CostPrice   decimal.Decimal `db:"cost_price"`   // set by platform admin
	FloorPrice  decimal.Decimal `db:"floor_price"`  // minimum sell price (>= cost)
	SellPrice   decimal.Decimal `db:"sell_price"`   // reseller's sell price to users
	IsActive    bool            `db:"is_active"`
	CreatedAt   time.Time       `db:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at"`
}

// ResellerWebhook stores a reseller's webhook endpoint.
type ResellerWebhook struct {
	ID         uuid.UUID  `db:"id"`
	ResellerID uuid.UUID  `db:"reseller_id"`
	URL        string     `db:"url"`
	Secret     string     `db:"secret"`     // HMAC signing secret
	Events     []string   `db:"-"`          // JSON stored separately
	IsActive   bool       `db:"is_active"`
	CreatedAt  time.Time  `db:"created_at"`
}

// ResellerAPIKey is a hashed API key for reseller programmatic access.
type ResellerAPIKey struct {
	ID          uuid.UUID  `db:"id"`
	ResellerID  uuid.UUID  `db:"reseller_id"`
	Name        string     `db:"name"`
	KeyHash     string     `db:"key_hash"`   // SHA256
	KeyPrefix   string     `db:"key_prefix"` // first 8 chars for display
	Scopes      []string   `db:"-"`
	LastUsedAt  *time.Time `db:"last_used_at"`
	ExpiresAt   *time.Time `db:"expires_at"`
	RevokedAt   *time.Time `db:"revoked_at"`
	CreatedAt   time.Time  `db:"created_at"`
}

// SubAccount is a user managed under a reseller.
type SubAccount struct {
	ID          uuid.UUID       `db:"id"`
	ResellerID  uuid.UUID       `db:"reseller_id"`
	UserID      uuid.UUID       `db:"user_id"`
	CreditLimit decimal.Decimal `db:"credit_limit"`
	CreatedAt   time.Time       `db:"created_at"`
}

// DashboardStats is aggregated stats for a reseller's dashboard.
type DashboardStats struct {
	TotalUsers      int             `json:"total_users"`
	TotalOrders     int             `json:"total_orders"`
	TotalRevenue    decimal.Decimal `json:"total_revenue"`
	TotalCost       decimal.Decimal `json:"total_cost"`
	GrossMargin     decimal.Decimal `json:"gross_margin"`
	WalletBalance   decimal.Decimal `json:"wallet_balance"`
	ActiveProxies   int             `json:"active_proxies"`
	ActiveVPS       int             `json:"active_vps"`
}

// ─── Repository Interfaces ────────────────────────────────────────────────────

type IResellerRepository interface {
	Create(ctx context.Context, r *ResellerAccount) error
	GetByID(ctx context.Context, id uuid.UUID) (*ResellerAccount, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*ResellerAccount, error)
	List(ctx context.Context, status string, offset, limit int) ([]*ResellerAccount, int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, approvedAt, suspendedAt *time.Time, reason string) error
	UpdateCreditLimit(ctx context.Context, id uuid.UUID, limit decimal.Decimal) error
}

type IPricingRepository interface {
	Upsert(ctx context.Context, p *PricingOverride) error
	GetByResellerAndProduct(ctx context.Context, resellerID, productID uuid.UUID) (*PricingOverride, error)
	ListByReseller(ctx context.Context, resellerID uuid.UUID) ([]*PricingOverride, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type IAPIKeyRepository interface {
	Create(ctx context.Context, k *ResellerAPIKey) error
	GetByHash(ctx context.Context, hash string) (*ResellerAPIKey, error)
	ListByReseller(ctx context.Context, resellerID uuid.UUID) ([]*ResellerAPIKey, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}

type ISubAccountRepository interface {
	Create(ctx context.Context, s *SubAccount) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*SubAccount, error)
	ListByReseller(ctx context.Context, resellerID uuid.UUID, offset, limit int) ([]*SubAccount, int, error)
	UpdateCreditLimit(ctx context.Context, id uuid.UUID, limit decimal.Decimal) error
}

type IWebhookRepository interface {
	Create(ctx context.Context, w *ResellerWebhook) error
	GetByID(ctx context.Context, id uuid.UUID) (*ResellerWebhook, error)
	ListByReseller(ctx context.Context, resellerID uuid.UUID) ([]*ResellerWebhook, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type IEventPublisher interface {
	PublishCreated(ctx context.Context, r *ResellerAccount) error
	PublishApproved(ctx context.Context, resellerID uuid.UUID) error
	PublishSuspended(ctx context.Context, resellerID uuid.UUID, reason string) error
	PublishPricingUpdated(ctx context.Context, resellerID, productID uuid.UUID, newPrice decimal.Decimal) error
}
