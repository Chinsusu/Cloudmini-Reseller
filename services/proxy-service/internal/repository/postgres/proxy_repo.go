package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pvp/proxy-service/internal/domain"
)

// OrderRepository implements domain.IOrderRepository.
type OrderRepository struct{ db *sqlx.DB }

func NewOrderRepository(db *sqlx.DB) *OrderRepository { return &OrderRepository{db: db} }

func (r *OrderRepository) Create(ctx context.Context, o *domain.Order) error {
	q := `INSERT INTO proxy.orders
		(id,order_number,user_id,reseller_id,product_id,provider_id,status,quantity,
		 unit_price,total_amount,idempotency_key,request_id,created_at,updated_at)
		VALUES (:id,:order_number,:user_id,:reseller_id,:product_id,:provider_id,:status,
		 :quantity,:unit_price,:total_amount,:idempotency_key,:request_id,:created_at,:updated_at)`
	if _, err := r.db.NamedExecContext(ctx, q, o); err != nil {
		return fmt.Errorf("OrderRepository.Create: %w", err)
	}
	return nil
}

func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	var o domain.Order
	if err := r.db.GetContext(ctx, &o, orderSelectCOALESCE+` WHERE id=$1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("OrderRepository.GetByID: %w", err)
	}
	return &o, nil
}

// orderSelectCOALESCE is the safe SELECT for order rows — handles nullable text columns.
const orderSelectCOALESCE = `SELECT
	id, order_number, user_id, reseller_id, product_id, provider_id, status, quantity,
	unit_price, total_amount,
	COALESCE(provider_order_id,'') AS provider_order_id,
	COALESCE(credentials,'')       AS credentials,
	activated_at, expires_at, cancelled_at,
	COALESCE(cancel_reason,'')     AS cancel_reason,
	idempotency_key, request_id, created_at, updated_at
	FROM proxy.orders`

func (r *OrderRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error) {
	var o domain.Order
	if err := r.db.GetContext(ctx, &o, orderSelectCOALESCE+` WHERE idempotency_key=$1`, key); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("OrderRepository.GetByIdempotencyKey: %w", err)
	}
	return &o, nil
}

func (r *OrderRepository) GetByProviderOrderID(ctx context.Context, providerOrderID string) (*domain.Order, error) {
	var o domain.Order
	if err := r.db.GetContext(ctx, &o,
		orderSelectCOALESCE+` WHERE provider_order_id=$1 LIMIT 1`, providerOrderID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("OrderRepository.GetByProviderOrderID: %w", err)
	}
	return &o, nil
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if _, err := r.db.ExecContext(ctx,
		`UPDATE proxy.orders SET status=$1, updated_at=NOW() WHERE id=$2`, status, id,
	); err != nil {
		return fmt.Errorf("OrderRepository.UpdateStatus: %w", err)
	}
	return nil
}

// UpdateAfterPurchase stores the provider order ID and optionally credentials/timestamps.
// For async providers (Proxy-Cheap), credentials, activatedAt, expiresAt may be zero values
// on the initial call; they are populated by the webhook usecase upon activation.
func (r *OrderRepository) UpdateAfterPurchase(ctx context.Context, id uuid.UUID, providerOrderID, credentials string, activatedAt, expiresAt *time.Time) error {
	if _, err := r.db.ExecContext(ctx,
		`UPDATE proxy.orders
		 SET provider_order_id=$1,
		     credentials=NULLIF($2,''),
		     activated_at=$3,
		     expires_at=$4,
		     status=CASE WHEN $2='' THEN status ELSE 'active' END,
		     updated_at=NOW()
		 WHERE id=$5`,
		providerOrderID, credentials, activatedAt, expiresAt, id,
	); err != nil {
		return fmt.Errorf("OrderRepository.UpdateAfterPurchase: %w", err)
	}
	return nil
}

func (r *OrderRepository) ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Order, int, error) {
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM proxy.orders WHERE user_id=$1`, userID); err != nil {
		return nil, 0, fmt.Errorf("OrderRepository.ListByUser: count: %w", err)
	}
	var orders []*domain.Order
	if err := r.db.SelectContext(ctx, &orders,
		orderSelectCOALESCE+` WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	); err != nil {
		return nil, 0, fmt.Errorf("OrderRepository.ListByUser: select: %w", err)
	}
	return orders, total, nil
}

func (r *OrderRepository) ListExpiring(ctx context.Context, within time.Duration) ([]*domain.Order, error) {
	cutoff := time.Now().Add(within)
	var orders []*domain.Order
	if err := r.db.SelectContext(ctx, &orders,
		orderSelectCOALESCE+` WHERE status='active' AND expires_at IS NOT NULL AND expires_at < $1`, cutoff,
	); err != nil {
		return nil, fmt.Errorf("OrderRepository.ListExpiring: %w", err)
	}
	return orders, nil
}

// ─── ProductRepository ─────────────────────────────────────────────────────────

type ProductRepository struct{ db *sqlx.DB }

func NewProductRepository(db *sqlx.DB) *ProductRepository { return &ProductRepository{db: db} }

func (r *ProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	var p domain.Product
	if err := r.db.GetContext(ctx, &p, `SELECT * FROM proxy.products WHERE id=$1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrProductNotFound
		}
		return nil, fmt.Errorf("ProductRepository.GetByID: %w", err)
	}
	return &p, nil
}

func (r *ProductRepository) List(ctx context.Context, proxyType, protocol, location string, offset, limit int) ([]*domain.Product, int, error) {
	var total int
	_ = r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM proxy.products WHERE is_active=true`)
	var products []*domain.Product
	err := r.db.SelectContext(ctx, &products,
		`SELECT * FROM proxy.products WHERE is_active=true ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	return products, total, err
}

func (r *ProductRepository) AdminList(ctx context.Context, offset, limit int) ([]*domain.Product, int, error) {
	var total int
	_ = r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM proxy.products`)
	var products []*domain.Product
	err := r.db.SelectContext(ctx, &products,
		`SELECT * FROM proxy.products ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	return products, total, err
}

func (r *ProductRepository) Create(ctx context.Context, p *domain.Product) error {
	q := `INSERT INTO proxy.products
		(id,provider_id,name,proxy_type,protocol,location,duration_days,bandwidth_gb,base_cost,is_active,created_at)
		VALUES (:id,:provider_id,:name,:proxy_type,:protocol,:location,:duration_days,:bandwidth_gb,:base_cost,:is_active,NOW())`
	if _, err := r.db.NamedExecContext(ctx, q, p); err != nil {
		return fmt.Errorf("ProductRepository.Create: %w", err)
	}
	return nil
}

func (r *ProductRepository) Update(ctx context.Context, p *domain.Product) error {
	q := `UPDATE proxy.products SET
		name=:name, proxy_type=:proxy_type, protocol=:protocol, location=:location,
		duration_days=:duration_days, bandwidth_gb=:bandwidth_gb, base_cost=:base_cost
		WHERE id=:id`
	if _, err := r.db.NamedExecContext(ctx, q, p); err != nil {
		return fmt.Errorf("ProductRepository.Update: %w", err)
	}
	return nil
}

func (r *ProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM proxy.products WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("ProductRepository.Delete: %w", err)
	}
	return nil
}

func (r *ProductRepository) ToggleActive(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE proxy.products SET is_active = NOT is_active WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("ProductRepository.ToggleActive: %w", err)
	}
	return nil
}

// ─── ProviderRepository ────────────────────────────────────────────────────────

type ProviderRepository struct{ db *sqlx.DB }

func NewProviderRepository(db *sqlx.DB) *ProviderRepository { return &ProviderRepository{db: db} }

func (r *ProviderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Provider, error) {
	var p domain.Provider
	if err := r.db.GetContext(ctx, &p, `SELECT * FROM proxy.providers WHERE id=$1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrProviderNotFound
		}
		return nil, fmt.Errorf("ProviderRepository.GetByID: %w", err)
	}
	return &p, nil
}

func (r *ProviderRepository) ListActive(ctx context.Context) ([]*domain.Provider, error) {
	var providers []*domain.Provider
	if err := r.db.SelectContext(ctx, &providers,
		`SELECT * FROM proxy.providers WHERE is_active=true ORDER BY priority DESC`,
	); err != nil {
		return nil, fmt.Errorf("ProviderRepository.ListActive: %w", err)
	}
	return providers, nil
}
