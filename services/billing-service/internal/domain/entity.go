// Package domain contains core entities, repository interfaces, and errors
// for billing-service.
package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ─── Entities ────────────────────────────────────────────────────────────────

// TransactionType constants.
const (
	TxnDeposit      = "deposit"
	TxnOrderCharge  = "order_charge"
	TxnOrderRefund  = "order_refund"
	TxnHold         = "hold"
	TxnHoldRelease  = "hold_release"
	TxnHoldConfirm  = "hold_confirm"
	TxnAdjustment   = "adjustment"
)

// Wallet holds user balance and hold amount.
type Wallet struct {
	ID                  uuid.UUID       `db:"id"`
	UserID              uuid.UUID       `db:"user_id"`
	Balance             decimal.Decimal `db:"balance"`
	HoldAmount          decimal.Decimal `db:"hold_amount"`
	Currency            string          `db:"currency"`
	LowBalanceThreshold decimal.Decimal `db:"low_balance_threshold"`
	LastAlertAt         *time.Time      `db:"last_alert_at"`
	CreatedAt           time.Time       `db:"created_at"`
	UpdatedAt           time.Time       `db:"updated_at"`
}

// AvailableBalance returns balance minus hold.
func (w *Wallet) AvailableBalance() decimal.Decimal {
	return w.Balance.Sub(w.HoldAmount)
}

// Transaction records every balance change.
type Transaction struct {
	ID            uuid.UUID       `db:"id"`
	TxnNumber     string          `db:"txn_number"`
	WalletID      uuid.UUID       `db:"wallet_id"`
	UserID        uuid.UUID       `db:"user_id"`
	Type          string          `db:"type"`
	Amount        decimal.Decimal `db:"amount"`
	BalanceBefore decimal.Decimal `db:"balance_before"`
	BalanceAfter  decimal.Decimal `db:"balance_after"`
	ReferenceType string          `db:"reference_type"`
	ReferenceID   *uuid.UUID      `db:"reference_id"`
	Description   string          `db:"description"`
	Metadata      map[string]any  `db:"-"`
	RequestID     *uuid.UUID      `db:"request_id"`
	CreatedAt     time.Time       `db:"created_at"`
}

// Payment status constants.
const (
	PaymentPending    = "pending"
	PaymentProcessing = "processing"
	PaymentCompleted  = "completed"
	PaymentFailed     = "failed"
	PaymentRefunded   = "refunded"
)

// Payment represents a deposit via external gateway.
type Payment struct {
	ID              uuid.UUID       `db:"id"`
	PaymentNumber   string          `db:"payment_number"`
	UserID          uuid.UUID       `db:"user_id"`
	WalletID        uuid.UUID       `db:"wallet_id"`
	Gateway         string          `db:"gateway"`
	GatewayTxnID    string          `db:"gateway_txn_id"`
	Amount          decimal.Decimal `db:"amount"`
	Currency        string          `db:"currency"`
	Status          string          `db:"status"`
	CompletedAt     *time.Time      `db:"completed_at"`
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
}

// PricingRule defines a markup pricing rule.
type PricingRule struct {
	ID          uuid.UUID       `db:"id"`
	ResellerID  *uuid.UUID      `db:"reseller_id"`
	ProductType string          `db:"product_type"`
	ProductID   *uuid.UUID      `db:"product_id"`
	MarkupType  string          `db:"markup_type"` // percentage|fixed
	MarkupValue decimal.Decimal `db:"markup_value"`
	MinPrice    *decimal.Decimal `db:"min_price"`
	IsActive    bool            `db:"is_active"`
	CreatedAt   time.Time       `db:"created_at"`
}

// ─── Repository Interfaces ────────────────────────────────────────────────────

// IWalletRepository defines wallet persistence operations.
type IWalletRepository interface {
	Create(ctx context.Context, w *Wallet) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*Wallet, error)
	GetByUserIDForUpdate(ctx context.Context, userID uuid.UUID) (*Wallet, error) // SELECT FOR UPDATE
	UpdateBalance(ctx context.Context, walletID uuid.UUID, balance, holdAmount decimal.Decimal) error
	List(ctx context.Context, offset, limit int) ([]*Wallet, int, error)
}

// ITransactionRepository defines transaction persistence operations.
type ITransactionRepository interface {
	Create(ctx context.Context, t *Transaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*Transaction, error)
	ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*Transaction, int, error)
}

// IPaymentRepository defines payment persistence operations.
type IPaymentRepository interface {
	Create(ctx context.Context, p *Payment) error
	GetByID(ctx context.Context, id uuid.UUID) (*Payment, error)
	GetByGatewayTxnID(ctx context.Context, gatewayTxnID string) (*Payment, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status, gatewayTxnID string) error
}

// IPricingRepository defines pricing rule persistence.
type IPricingRepository interface {
	GetRule(ctx context.Context, resellerID *uuid.UUID, productType string, productID *uuid.UUID) (*PricingRule, error)
	UpsertRule(ctx context.Context, rule *PricingRule) error
	ListByReseller(ctx context.Context, resellerID uuid.UUID) ([]*PricingRule, error)
}

// IEventPublisher publishes billing events.
type IEventPublisher interface {
	PublishCharged(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, ref string) error
	PublishDepositCompleted(ctx context.Context, userID uuid.UUID, amount decimal.Decimal) error
	PublishWalletLow(ctx context.Context, userID uuid.UUID, balance decimal.Decimal) error
	PublishWalletEmpty(ctx context.Context, userID uuid.UUID) error
}

// ITxRunner executes a function inside a DB transaction.
type ITxRunner interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
