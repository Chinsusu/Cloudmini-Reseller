package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pvp/billing-service/internal/domain"
	"github.com/shopspring/decimal"
)

// WalletRepository implements domain.IWalletRepository.
type WalletRepository struct{ db *sqlx.DB }

func NewWalletRepository(db *sqlx.DB) *WalletRepository { return &WalletRepository{db: db} }

func (r *WalletRepository) Create(ctx context.Context, w *domain.Wallet) error {
	q := `INSERT INTO billing.wallets (id,user_id,balance,hold_amount,currency,low_balance_threshold,created_at,updated_at)
		  VALUES (:id,:user_id,:balance,:hold_amount,:currency,:low_balance_threshold,:created_at,:updated_at)`
	if _, err := r.db.NamedExecContext(ctx, q, w); err != nil {
		return fmt.Errorf("WalletRepository.Create: %w", err)
	}
	return nil
}

func (r *WalletRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Wallet, error) {
	var w domain.Wallet
	if err := r.db.GetContext(ctx, &w, `SELECT * FROM billing.wallets WHERE user_id=$1`, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWalletNotFound
		}
		return nil, fmt.Errorf("WalletRepository.GetByUserID: %w", err)
	}
	return &w, nil
}

func (r *WalletRepository) GetByUserIDForUpdate(ctx context.Context, userID uuid.UUID) (*domain.Wallet, error) {
	var w domain.Wallet
	if err := r.db.GetContext(ctx, &w, `SELECT * FROM billing.wallets WHERE user_id=$1 FOR UPDATE`, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWalletNotFound
		}
		return nil, fmt.Errorf("WalletRepository.GetByUserIDForUpdate: %w", err)
	}
	return &w, nil
}

func (r *WalletRepository) UpdateBalance(ctx context.Context, walletID uuid.UUID, balance, holdAmount decimal.Decimal) error {
	q := `UPDATE billing.wallets SET balance=$1, hold_amount=$2, updated_at=NOW() WHERE id=$3`
	if _, err := r.db.ExecContext(ctx, q, balance, holdAmount, walletID); err != nil {
		return fmt.Errorf("WalletRepository.UpdateBalance: %w", err)
	}
	return nil
}

func (r *WalletRepository) List(ctx context.Context, offset, limit int) ([]*domain.Wallet, int, error) {
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM billing.wallets`); err != nil {
		return nil, 0, fmt.Errorf("WalletRepository.List: count: %w", err)
	}
	var wallets []*domain.Wallet
	if err := r.db.SelectContext(ctx, &wallets, `SELECT * FROM billing.wallets ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset); err != nil {
		return nil, 0, fmt.Errorf("WalletRepository.List: select: %w", err)
	}
	return wallets, total, nil
}

// TxRunner implements domain.ITxRunner using sqlx transactions.
type TxRunner struct{ db *sqlx.DB }

func NewTxRunner(db *sqlx.DB) *TxRunner { return &TxRunner{db: db} }

// txKey is the context key to pass an in-progress transaction.
type txKey struct{}

func (t *TxRunner) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := t.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("TxRunner.RunInTx: begin: %w", err)
	}
	txCtx := context.WithValue(ctx, txKey{}, tx)
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("TxRunner.RunInTx: commit: %w", err)
	}
	return nil
}

// ─── TransactionRepository ────────────────────────────────────────────────────

type TransactionRepository struct{ db *sqlx.DB }

func NewTransactionRepository(db *sqlx.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(ctx context.Context, t *domain.Transaction) error {
	// Use tx from context if present
	q := `INSERT INTO billing.transactions
		(id,txn_number,wallet_id,user_id,type,amount,balance_before,balance_after,
		 reference_type,reference_id,description,request_id,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`
	db := txOrDB(ctx, r.db)
	if _, err := db.ExecContext(ctx, q,
		t.ID, t.TxnNumber, t.WalletID, t.UserID, t.Type, t.Amount,
		t.BalanceBefore, t.BalanceAfter, t.ReferenceType, t.ReferenceID,
		t.Description, t.RequestID, t.CreatedAt,
	); err != nil {
		return fmt.Errorf("TransactionRepository.Create: %w", err)
	}
	return nil
}

func (r *TransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	var t domain.Transaction
	if err := r.db.GetContext(ctx, &t, `SELECT * FROM billing.transactions WHERE id=$1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("TransactionRepository.GetByID: %w", err)
	}
	return &t, nil
}

func (r *TransactionRepository) ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Transaction, int, error) {
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM billing.transactions WHERE user_id=$1`, userID); err != nil {
		return nil, 0, fmt.Errorf("TransactionRepository.ListByUser: count: %w", err)
	}
	var txns []*domain.Transaction
	q := `SELECT * FROM billing.transactions WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	if err := r.db.SelectContext(ctx, &txns, q, userID, limit, offset); err != nil {
		return nil, 0, fmt.Errorf("TransactionRepository.ListByUser: select: %w", err)
	}
	return txns, total, nil
}

// ─── PaymentRepository ────────────────────────────────────────────────────────

type PaymentRepository struct{ db *sqlx.DB }

func NewPaymentRepository(db *sqlx.DB) *PaymentRepository { return &PaymentRepository{db: db} }

func (r *PaymentRepository) Create(ctx context.Context, p *domain.Payment) error {
	q := `INSERT INTO billing.payments (id,payment_number,user_id,wallet_id,gateway,amount,currency,status,created_at,updated_at)
		  VALUES (:id,:payment_number,:user_id,:wallet_id,:gateway,:amount,:currency,:status,:created_at,:updated_at)`
	if _, err := r.db.NamedExecContext(ctx, q, p); err != nil {
		return fmt.Errorf("PaymentRepository.Create: %w", err)
	}
	return nil
}

func (r *PaymentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	var p domain.Payment
	if err := r.db.GetContext(ctx, &p, `SELECT * FROM billing.payments WHERE id=$1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("PaymentRepository.GetByID: %w", err)
	}
	return &p, nil
}

func (r *PaymentRepository) GetByGatewayTxnID(ctx context.Context, gatewayTxnID string) (*domain.Payment, error) {
	var p domain.Payment
	if err := r.db.GetContext(ctx, &p, `SELECT * FROM billing.payments WHERE gateway_txn_id=$1`, gatewayTxnID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("PaymentRepository.GetByGatewayTxnID: %w", err)
	}
	return &p, nil
}

func (r *PaymentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status, gatewayTxnID string) error {
	q := `UPDATE billing.payments SET status=$1, gateway_txn_id=$2, completed_at=NOW(), updated_at=NOW() WHERE id=$3`
	if _, err := r.db.ExecContext(ctx, q, status, gatewayTxnID, id); err != nil {
		return fmt.Errorf("PaymentRepository.UpdateStatus: %w", err)
	}
	return nil
}

// ─── PricingRepository ────────────────────────────────────────────────────────

type PricingRepository struct{ db *sqlx.DB }

func NewPricingRepository(db *sqlx.DB) *PricingRepository { return &PricingRepository{db: db} }

func (r *PricingRepository) GetRule(ctx context.Context, resellerID *uuid.UUID, productType string, productID *uuid.UUID) (*domain.PricingRule, error) {
	var rule domain.PricingRule
	var err error
	if resellerID == nil && productID == nil {
		err = r.db.GetContext(ctx, &rule, `SELECT * FROM billing.pricing_rules WHERE reseller_id IS NULL AND product_type=$1 AND product_id IS NULL AND is_active=true LIMIT 1`, productType)
	} else if resellerID == nil {
		err = r.db.GetContext(ctx, &rule, `SELECT * FROM billing.pricing_rules WHERE reseller_id IS NULL AND product_type=$1 AND product_id=$2 AND is_active=true LIMIT 1`, productType, productID)
	} else if productID == nil {
		err = r.db.GetContext(ctx, &rule, `SELECT * FROM billing.pricing_rules WHERE reseller_id=$1 AND product_type=$2 AND product_id IS NULL AND is_active=true LIMIT 1`, resellerID, productType)
	} else {
		err = r.db.GetContext(ctx, &rule, `SELECT * FROM billing.pricing_rules WHERE reseller_id=$1 AND product_type=$2 AND product_id=$3 AND is_active=true LIMIT 1`, resellerID, productType, productID)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPricingRuleNotFound
		}
		return nil, fmt.Errorf("PricingRepository.GetRule: %w", err)
	}
	return &rule, nil
}

func (r *PricingRepository) UpsertRule(ctx context.Context, rule *domain.PricingRule) error {
	q := `INSERT INTO billing.pricing_rules (id,reseller_id,product_type,product_id,markup_type,markup_value,min_price,is_active,created_at)
		  VALUES (:id,:reseller_id,:product_type,:product_id,:markup_type,:markup_value,:min_price,:is_active,:created_at)
		  ON CONFLICT (id) DO UPDATE SET markup_type=EXCLUDED.markup_type, markup_value=EXCLUDED.markup_value, min_price=EXCLUDED.min_price`
	if _, err := r.db.NamedExecContext(ctx, q, rule); err != nil {
		return fmt.Errorf("PricingRepository.UpsertRule: %w", err)
	}
	return nil
}

func (r *PricingRepository) ListByReseller(ctx context.Context, resellerID uuid.UUID) ([]*domain.PricingRule, error) {
	var rules []*domain.PricingRule
	if err := r.db.SelectContext(ctx, &rules, `SELECT * FROM billing.pricing_rules WHERE reseller_id=$1 AND is_active=true`, resellerID); err != nil {
		return nil, fmt.Errorf("PricingRepository.ListByReseller: %w", err)
	}
	return rules, nil
}

// txOrDB returns the transaction from context if available, else the plain DB.
func txOrDB(ctx context.Context, db *sqlx.DB) sqlx.ExecerContext {
	if tx, ok := ctx.Value(txKey{}).(*sqlx.Tx); ok {
		return tx
	}
	return db
}
