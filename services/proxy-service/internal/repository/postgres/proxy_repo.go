package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pvp/proxy-service/internal/domain"
	"github.com/shopspring/decimal"
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
	unit_price, total_amount, custom_price,
	COALESCE(provider_order_id,'') AS provider_order_id,
	COALESCE(credentials,'')       AS credentials,
	activated_at, expires_at, custom_expires_at, cancelled_at,
	COALESCE(cancel_reason,'')     AS cancel_reason,
	COALESCE(admin_note,'')        AS admin_note,
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

// UpdateOrder sets custom_price, custom_expires_at and admin_note for an order.
func (r *OrderRepository) UpdateOrder(ctx context.Context, id uuid.UUID, customPrice *decimal.Decimal, customExpiresAt *time.Time, adminNote string) error {
	if _, err := r.db.ExecContext(ctx,
		`UPDATE proxy.orders SET custom_price=$1, custom_expires_at=$2, admin_note=NULLIF($3,''), updated_at=NOW() WHERE id=$4`,
		customPrice, customExpiresAt, adminNote, id,
	); err != nil {
		return fmt.Errorf("OrderRepository.UpdateOrder: %w", err)
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
	// Exclude 'failed' orders — they are logged in order_events but not shown to users
	const excludeFailed = ` WHERE user_id=$1 AND status != 'failed'`
	if err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM proxy.orders`+excludeFailed, userID,
	); err != nil {
		return nil, 0, fmt.Errorf("OrderRepository.ListByUser: count: %w", err)
	}
	var orders []*domain.Order
	if err := r.db.SelectContext(ctx, &orders,
		orderSelectCOALESCE+excludeFailed+` ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
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

// ListExpiredActive returns active orders whose effective expiry (COALESCE(custom_expires_at, expires_at))
// has already passed. These are candidates for the first expiry step: stop + move to 'expired'.
func (r *OrderRepository) ListExpiredActive(ctx context.Context) ([]*domain.Order, error) {
	var orders []*domain.Order
	if err := r.db.SelectContext(ctx, &orders,
		orderSelectCOALESCE+`
		WHERE status = 'active'
		  AND COALESCE(custom_expires_at, expires_at) IS NOT NULL
		  AND COALESCE(custom_expires_at, expires_at) < NOW()`,
	); err != nil {
		return nil, fmt.Errorf("OrderRepository.ListExpiredActive: %w", err)
	}
	return orders, nil
}

// ListExpiredGrace returns expired orders whose effective expiry + grace period has passed.
// These are candidates for permanent deletion from the provider.
func (r *OrderRepository) ListExpiredGrace(ctx context.Context, grace time.Duration) ([]*domain.Order, error) {
	cutoff := time.Now().Add(-grace) // grace ago = must have expired before this time
	var orders []*domain.Order
	if err := r.db.SelectContext(ctx, &orders,
		orderSelectCOALESCE+`
		WHERE status = 'expired'
		  AND COALESCE(custom_expires_at, expires_at) IS NOT NULL
		  AND COALESCE(custom_expires_at, expires_at) < $1`,
		cutoff,
	); err != nil {
		return nil, fmt.Errorf("OrderRepository.ListExpiredGrace: %w", err)
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

func (r *ProviderRepository) ListAll(ctx context.Context) ([]*domain.Provider, error) {
	var providers []*domain.Provider
	if err := r.db.SelectContext(ctx, &providers,
		`SELECT * FROM proxy.providers ORDER BY priority DESC, name`,
	); err != nil {
		return nil, fmt.Errorf("ProviderRepository.ListAll: %w", err)
	}
	return providers, nil
}

func (r *ProviderRepository) ListByAdapterType(ctx context.Context, adapterType string) ([]*domain.Provider, error) {
	var providers []*domain.Provider
	if err := r.db.SelectContext(ctx, &providers,
		`SELECT * FROM proxy.providers WHERE adapter_type=$1 AND is_active=true ORDER BY priority DESC`,
		adapterType,
	); err != nil {
		return nil, fmt.Errorf("ProviderRepository.ListByAdapterType: %w", err)
	}
	return providers, nil
}

func (r *ProviderRepository) Create(ctx context.Context, p *domain.Provider) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO proxy.providers (id, name, display_name, adapter_type, config, is_active, priority)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		p.ID, p.Name, p.DisplayName, p.AdapterType, p.Config, p.IsActive, p.Priority,
	)
	if err != nil {
		return fmt.Errorf("ProviderRepository.Create: %w", err)
	}
	return nil
}

func (r *ProviderRepository) Update(ctx context.Context, p *domain.Provider) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE proxy.providers
		 SET name=$2, display_name=$3, adapter_type=$4, config=$5, is_active=$6, priority=$7, updated_at=NOW()
		 WHERE id=$1`,
		p.ID, p.Name, p.DisplayName, p.AdapterType, p.Config, p.IsActive, p.Priority,
	)
	if err != nil {
		return fmt.Errorf("ProviderRepository.Update: %w", err)
	}
	return nil
}

func (r *ProviderRepository) ToggleActive(ctx context.Context, id uuid.UUID, active bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE proxy.providers SET is_active=$2, updated_at=NOW() WHERE id=$1`,
		id, active,
	)
	if err != nil {
		return fmt.Errorf("ProviderRepository.ToggleActive: %w", err)
	}
	return nil
}

func (r *ProviderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM proxy.providers WHERE id=$1`, id,
	)
	if err != nil {
		return fmt.Errorf("ProviderRepository.Delete: %w", err)
	}
	return nil
}


// ─── OrderEventRepository ──────────────────────────────────────────────────────

// OrderEventRepository implements domain.IOrderEventRepository using PostgreSQL.
type OrderEventRepository struct{ db *sqlx.DB }

func NewOrderEventRepository(db *sqlx.DB) *OrderEventRepository {
	return &OrderEventRepository{db: db}
}

// Log inserts a new event row for the given order.
func (r *OrderEventRepository) Log(ctx context.Context, orderID uuid.UUID, eventType string, payload map[string]any) error {
	p := []byte("{}")
	if payload != nil {
		if b, err := json.Marshal(payload); err == nil {
			p = b
		}
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO proxy.order_events (order_id, event_type, payload) VALUES ($1, $2, $3)`,
		orderID, eventType, p,
	)
	if err != nil {
		return fmt.Errorf("OrderEventRepository.Log: %w", err)
	}
	return nil
}

// ListByOrder returns all events for an order in chronological order.
func (r *OrderEventRepository) ListByOrder(ctx context.Context, orderID uuid.UUID) ([]*domain.OrderEvent, error) {
	var events []*domain.OrderEvent
	if err := r.db.SelectContext(ctx, &events,
		`SELECT id, order_id, event_type, payload, created_at
		 FROM proxy.order_events WHERE order_id=$1 ORDER BY created_at ASC`,
		orderID,
	); err != nil {
		return nil, fmt.Errorf("OrderEventRepository.ListByOrder: %w", err)
	}
	return events, nil
}
