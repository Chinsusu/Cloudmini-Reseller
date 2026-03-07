package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pvp/reseller-service/internal/domain"
	"github.com/shopspring/decimal"
)

// ─── ResellerRepository ───────────────────────────────────────────────────────

type ResellerRepository struct{ db *sqlx.DB }

func NewResellerRepository(db *sqlx.DB) *ResellerRepository { return &ResellerRepository{db: db} }

func (r *ResellerRepository) Create(ctx context.Context, res *domain.ResellerAccount) error {
	q := `INSERT INTO resellers.accounts
		(id,user_id,company_name,email,phone,address,tax_id,status,commission_pct,created_at,updated_at)
		VALUES (:id,:user_id,:company_name,:email,:phone,:address,:tax_id,:status,:commission_pct,:created_at,:updated_at)`
	if _, err := r.db.NamedExecContext(ctx, q, res); err != nil {
		return fmt.Errorf("ResellerRepository.Create: %w", err)
	}
	return nil
}

func (r *ResellerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ResellerAccount, error) {
	var res domain.ResellerAccount
	if err := r.db.GetContext(ctx, &res, `SELECT * FROM resellers.accounts WHERE id=$1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrResellerNotFound
		}
		return nil, fmt.Errorf("ResellerRepository.GetByID: %w", err)
	}
	return &res, nil
}

func (r *ResellerRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.ResellerAccount, error) {
	var res domain.ResellerAccount
	if err := r.db.GetContext(ctx, &res, `SELECT * FROM resellers.accounts WHERE user_id=$1`, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrResellerNotFound
		}
		return nil, fmt.Errorf("ResellerRepository.GetByUserID: %w", err)
	}
	return &res, nil
}

func (r *ResellerRepository) List(ctx context.Context, status string, offset, limit int) ([]*domain.ResellerAccount, int, error) {
	var total int
	if status != "" {
		_ = r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM resellers.accounts WHERE status=$1`, status)
	} else {
		_ = r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM resellers.accounts`)
	}

	var resellers []*domain.ResellerAccount
	var err error
	if status != "" {
		err = r.db.SelectContext(ctx, &resellers,
			`SELECT * FROM resellers.accounts WHERE status=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			status, limit, offset)
	} else {
		err = r.db.SelectContext(ctx, &resellers,
			`SELECT * FROM resellers.accounts ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
			limit, offset)
	}
	return resellers, total, err
}

func (r *ResellerRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, approvedAt, suspendedAt *time.Time, reason string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE resellers.accounts SET status=$1, approved_at=$2, suspended_at=$3, suspend_reason=$4, updated_at=NOW() WHERE id=$5`,
		status, approvedAt, suspendedAt, reason, id)
	return err
}

func (r *ResellerRepository) UpdateCreditLimit(ctx context.Context, id uuid.UUID, limit decimal.Decimal) error {
	_, err := r.db.ExecContext(ctx, `UPDATE resellers.accounts SET credit_limit=$1, updated_at=NOW() WHERE id=$2`, limit, id)
	return err
}

// ─── PricingRepository ────────────────────────────────────────────────────────

type PricingRepository struct{ db *sqlx.DB }

func NewPricingRepository(db *sqlx.DB) *PricingRepository { return &PricingRepository{db: db} }

func (r *PricingRepository) Upsert(ctx context.Context, p *domain.PricingOverride) error {
	q := `INSERT INTO resellers.pricing_overrides
		(id,reseller_id,product_id,product_type,cost_price,floor_price,sell_price,is_active,created_at,updated_at)
		VALUES (:id,:reseller_id,:product_id,:product_type,:cost_price,:floor_price,:sell_price,:is_active,:created_at,:updated_at)
		ON CONFLICT (reseller_id, product_id) DO UPDATE SET
		sell_price=EXCLUDED.sell_price, floor_price=EXCLUDED.floor_price, updated_at=NOW()`
	if _, err := r.db.NamedExecContext(ctx, q, p); err != nil {
		return fmt.Errorf("PricingRepository.Upsert: %w", err)
	}
	return nil
}

func (r *PricingRepository) GetByResellerAndProduct(ctx context.Context, resellerID, productID uuid.UUID) (*domain.PricingOverride, error) {
	var p domain.PricingOverride
	if err := r.db.GetContext(ctx, &p,
		`SELECT * FROM resellers.pricing_overrides WHERE reseller_id=$1 AND product_id=$2`, resellerID, productID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPricingNotFound
		}
		return nil, fmt.Errorf("PricingRepository.Get: %w", err)
	}
	return &p, nil
}

func (r *PricingRepository) ListByReseller(ctx context.Context, resellerID uuid.UUID) ([]*domain.PricingOverride, error) {
	var pricing []*domain.PricingOverride
	err := r.db.SelectContext(ctx, &pricing,
		`SELECT * FROM resellers.pricing_overrides WHERE reseller_id=$1 AND is_active=true ORDER BY created_at DESC`, resellerID)
	return pricing, err
}

func (r *PricingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE resellers.pricing_overrides SET is_active=false WHERE id=$1`, id)
	return err
}

// ─── APIKeyRepository ─────────────────────────────────────────────────────────

type APIKeyRepository struct{ db *sqlx.DB }

func NewAPIKeyRepository(db *sqlx.DB) *APIKeyRepository { return &APIKeyRepository{db: db} }

func (r *APIKeyRepository) Create(ctx context.Context, k *domain.ResellerAPIKey) error {
	q := `INSERT INTO resellers.api_keys
		(id,reseller_id,name,key_hash,key_prefix,expires_at,created_at)
		VALUES (:id,:reseller_id,:name,:key_hash,:key_prefix,:expires_at,:created_at)`
	if _, err := r.db.NamedExecContext(ctx, q, k); err != nil {
		return fmt.Errorf("APIKeyRepository.Create: %w", err)
	}
	return nil
}

func (r *APIKeyRepository) GetByHash(ctx context.Context, hash string) (*domain.ResellerAPIKey, error) {
	var k domain.ResellerAPIKey
	if err := r.db.GetContext(ctx, &k,
		`SELECT * FROM resellers.api_keys WHERE key_hash=$1 AND revoked_at IS NULL`, hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("APIKeyRepository.GetByHash: %w", err)
	}
	return &k, nil
}

func (r *APIKeyRepository) ListByReseller(ctx context.Context, resellerID uuid.UUID) ([]*domain.ResellerAPIKey, error) {
	var keys []*domain.ResellerAPIKey
	err := r.db.SelectContext(ctx, &keys,
		`SELECT * FROM resellers.api_keys WHERE reseller_id=$1 ORDER BY created_at DESC`, resellerID)
	return keys, err
}

func (r *APIKeyRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE resellers.api_keys SET revoked_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE resellers.api_keys SET last_used_at=NOW() WHERE id=$1`, id)
	return err
}

// ─── SubAccountRepository ─────────────────────────────────────────────────────

type SubAccountRepository struct{ db *sqlx.DB }

func NewSubAccountRepository(db *sqlx.DB) *SubAccountRepository { return &SubAccountRepository{db: db} }

func (r *SubAccountRepository) Create(ctx context.Context, s *domain.SubAccount) error {
	q := `INSERT INTO resellers.sub_accounts (id,reseller_id,user_id,credit_limit,created_at)
		  VALUES (:id,:reseller_id,:user_id,:credit_limit,:created_at)`
	if _, err := r.db.NamedExecContext(ctx, q, s); err != nil {
		return fmt.Errorf("SubAccountRepository.Create: %w", err)
	}
	return nil
}

func (r *SubAccountRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.SubAccount, error) {
	var s domain.SubAccount
	if err := r.db.GetContext(ctx, &s, `SELECT * FROM resellers.sub_accounts WHERE user_id=$1`, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSubAccountNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *SubAccountRepository) ListByReseller(ctx context.Context, resellerID uuid.UUID, offset, limit int) ([]*domain.SubAccount, int, error) {
	var total int
	_ = r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM resellers.sub_accounts WHERE reseller_id=$1`, resellerID)
	var subs []*domain.SubAccount
	err := r.db.SelectContext(ctx, &subs,
		`SELECT * FROM resellers.sub_accounts WHERE reseller_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		resellerID, limit, offset)
	return subs, total, err
}

func (r *SubAccountRepository) UpdateCreditLimit(ctx context.Context, id uuid.UUID, limit decimal.Decimal) error {
	_, err := r.db.ExecContext(ctx, `UPDATE resellers.sub_accounts SET credit_limit=$1 WHERE id=$2`, limit, id)
	return err
}

// ─── WebhookRepository ────────────────────────────────────────────────────────

type WebhookRepository struct{ db *sqlx.DB }

func NewWebhookRepository(db *sqlx.DB) *WebhookRepository { return &WebhookRepository{db: db} }

func (r *WebhookRepository) Create(ctx context.Context, w *domain.ResellerWebhook) error {
	q := `INSERT INTO resellers.webhooks (id,reseller_id,url,secret,is_active,created_at)
		  VALUES (:id,:reseller_id,:url,:secret,:is_active,:created_at)`
	if _, err := r.db.NamedExecContext(ctx, q, w); err != nil {
		return fmt.Errorf("WebhookRepository.Create: %w", err)
	}
	return nil
}

func (r *WebhookRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ResellerWebhook, error) {
	var w domain.ResellerWebhook
	if err := r.db.GetContext(ctx, &w, `SELECT * FROM resellers.webhooks WHERE id=$1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWebhookNotFound
		}
		return nil, err
	}
	return &w, nil
}

func (r *WebhookRepository) ListByReseller(ctx context.Context, resellerID uuid.UUID) ([]*domain.ResellerWebhook, error) {
	var webhooks []*domain.ResellerWebhook
	err := r.db.SelectContext(ctx, &webhooks,
		`SELECT * FROM resellers.webhooks WHERE reseller_id=$1 AND is_active=true`, resellerID)
	return webhooks, err
}

func (r *WebhookRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE resellers.webhooks SET is_active=false WHERE id=$1`, id)
	return err
}
