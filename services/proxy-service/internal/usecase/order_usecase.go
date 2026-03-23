// Package usecase contains proxy-service business logic.
// order_usecase.go implements the 7-step Saga for proxy order creation.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	cryptopkg "github.com/pvp/pkg/crypto"
	"github.com/pvp/proxy-service/internal/domain"
	"github.com/pvp/proxy-service/internal/providers"
	"github.com/shopspring/decimal"
)

// BillingClient calls billing-service via internal HTTP.
// In production: use a proper gRPC client or internal HTTP client.
type BillingClient interface {
	Hold(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, refType string, refID uuid.UUID) error
	ConfirmHold(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, refType string, refID uuid.UUID, description string) error
	ReleaseHold(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, refType string, refID uuid.UUID, description string) error
	CalculatePrice(ctx context.Context, baseCost decimal.Decimal, productType string, productID uuid.UUID, resellerID *uuid.UUID) (decimal.Decimal, error)
}

// OrderUsecase handles the proxy order lifecycle.
type OrderUsecase struct {
	orderRepo     domain.IOrderRepository
	productRepo   domain.IProductRepository
	providerRepo  domain.IProviderRepository
	providerReg   *providers.Registry
	billingClient BillingClient
	cipher        *cryptopkg.Cipher
	eventPub      domain.IEventPublisher
	orderEvtRepo  domain.IOrderEventRepository
	logger        *slog.Logger
}

// NewOrderUsecase constructs OrderUsecase via DI.
func NewOrderUsecase(
	orderRepo domain.IOrderRepository,
	productRepo domain.IProductRepository,
	providerRepo domain.IProviderRepository,
	providerReg *providers.Registry,
	billingClient BillingClient,
	cipher *cryptopkg.Cipher,
	eventPub domain.IEventPublisher,
	orderEvtRepo domain.IOrderEventRepository,
	logger *slog.Logger,
) *OrderUsecase {
	return &OrderUsecase{
		orderRepo:     orderRepo,
		productRepo:   productRepo,
		providerRepo:  providerRepo,
		providerReg:   providerReg,
		billingClient: billingClient,
		cipher:        cipher,
		eventPub:      eventPub,
		orderEvtRepo:  orderEvtRepo,
		logger:        logger,
	}
}

// CreateOrderRequest is the input for the order Saga.
type CreateOrderRequest struct {
	UserID         uuid.UUID
	ResellerID     *uuid.UUID
	ProductID      uuid.UUID
	Quantity       int
	IdempotencyKey string
	RequestID      *uuid.UUID
	// Metadata passes provider-specific parameters (e.g. Proxy-Cheap service_id, plan_id).
	Metadata map[string]string
}

// CreateOrder runs the 7-step Saga:
//  1. Validate product
//  2. Calculate price via billing-service pricing engine
//  3. billing.Hold (compensate: ReleaseHold)
//  4. provider.Purchase (compensate: billing.ReleaseHold)
//  5. Encrypt credentials
//  6. Update order status → active
//  7. Publish order.proxy.fulfilled
func (u *OrderUsecase) CreateOrder(ctx context.Context, req CreateOrderRequest) (*domain.Order, error) {
	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	// Idempotency check
	if req.IdempotencyKey != "" {
		existing, err := u.orderRepo.GetByIdempotencyKey(ctx, req.IdempotencyKey)
		if err == nil {
			u.logger.InfoContext(ctx, "duplicate order request, returning existing", slog.String("order_id", existing.ID.String()))
			return existing, nil
		}
	}

	// ── Step 1: Validate product ───────────────────────────────────────────
	product, err := u.productRepo.GetByID(ctx, req.ProductID)
	if err != nil {
		return nil, fmt.Errorf("CreateOrder step1: %w", err)
	}
	if !product.IsActive {
		return nil, fmt.Errorf("CreateOrder step1: product is not active")
	}

	// ── Step 2: Calculate price ────────────────────────────────────────────
	unitPrice, err := u.billingClient.CalculatePrice(ctx, product.BaseCost, "proxy", product.ID, req.ResellerID)
	if err != nil {
		return nil, fmt.Errorf("CreateOrder step2: calculate price: %w", err)
	}
	totalAmount := unitPrice.Mul(decimal.NewFromInt(int64(req.Quantity)))

	// Create order in pending state
	order := &domain.Order{
		ID:             uuid.New(),
		OrderNumber:    fmt.Sprintf("PX-%s-%04X", time.Now().Format("060102"), uint16(time.Now().UnixNano()&0xFFFF)),
		UserID:         req.UserID,
		ResellerID:     req.ResellerID,
		ProductID:      req.ProductID,
		ProviderID:     product.ProviderID,
		Status:         domain.OrderPending,
		Quantity:       req.Quantity,
		UnitPrice:      unitPrice,
		TotalAmount:    totalAmount,
		IdempotencyKey: req.IdempotencyKey,
		RequestID:      req.RequestID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := u.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("CreateOrder: create order: %w", err)
	}
	// Log order.created event
	_ = u.orderEvtRepo.Log(ctx, order.ID, domain.EventOrderCreated, map[string]any{
		"product_id": order.ProductID.String(),
		"amount":     order.TotalAmount.String(),
	})

	// ── Step 3: Hold funds in billing ──────────────────────────────────────
	if err := u.billingClient.Hold(ctx, req.UserID, totalAmount, "proxy_order", order.ID); err != nil {
		_ = u.orderRepo.UpdateStatus(ctx, order.ID, domain.OrderFailed)
		_ = u.eventPub.PublishOrderFailed(ctx, order.ID, "insufficient_funds")
		return nil, fmt.Errorf("CreateOrder step3: hold: %w", err)
	}

	// ── Step 4: Purchase from provider ─────────────────────────────────────
	provider := u.providerReg.Get(product.ProviderID.String())
	if provider == nil {
		// Compensate: release hold
		_ = u.billingClient.ReleaseHold(ctx, req.UserID, totalAmount, "proxy_order", order.ID, "")
		_ = u.orderRepo.UpdateStatus(ctx, order.ID, domain.OrderFailed)
		_ = u.eventPub.PublishOrderFailed(ctx, order.ID, "no_provider_available")
		return nil, domain.ErrNoProviderAvailable
	}

	purchaseResult, err := provider.Purchase(ctx, providers.PurchaseRequest{
		ProductID: product.ID.String(),
		Quantity:  req.Quantity,
		OrderID:   order.ID.String(),
		Country:   product.Location,
		Protocol:  product.Protocol,
		Metadata:  req.Metadata,
	})
	if err != nil {
		// Compensate: release hold
		_ = u.billingClient.ReleaseHold(ctx, req.UserID, totalAmount, "proxy_order", order.ID, "")
		_ = u.orderRepo.UpdateStatus(ctx, order.ID, domain.OrderFailed)
		_ = u.eventPub.PublishOrderFailed(ctx, order.ID, "provider_purchase_failed")
		return nil, fmt.Errorf("CreateOrder step4: %w: %w", domain.ErrProviderPurchase, err)
	}

	// ── Step 5: Async or sync credential handling ──────────────────────────
	// Async providers (e.g. Proxy-Cheap) return nil Credentials.
	// In that case we store providerOrderID and set status=processing.
	// The webhook usecase will fulfill the order when the proxy becomes ACTIVE.
	if purchaseResult.Credentials == nil {
		// Store provider order ID, keep billing hold — webhook will confirm it
		if err := u.orderRepo.UpdateAfterPurchase(ctx, order.ID,
			purchaseResult.ProviderOrderID, "", nil, nil,
		); err != nil {
			_ = u.billingClient.ReleaseHold(ctx, req.UserID, totalAmount, "proxy_order", order.ID, "")
			_ = u.orderRepo.UpdateStatus(ctx, order.ID, domain.OrderFailed)
			return nil, fmt.Errorf("CreateOrder step5 (async): save provider order id: %w", err)
		}
		_ = u.orderRepo.UpdateStatus(ctx, order.ID, domain.OrderProcessing)
		order.Status = domain.OrderProcessing
		order.ProviderOrderID = purchaseResult.ProviderOrderID
		u.logger.InfoContext(ctx, "proxy order purchasing (async provider)",
			slog.String("order_id", order.ID.String()),
			slog.String("provider_order_id", purchaseResult.ProviderOrderID),
		)
		return order, nil
	}

	// Sync provider: encrypt credentials immediately
	credJSON, _ := json.Marshal(purchaseResult.Credentials)
	encryptedCreds, err := u.cipher.Encrypt(credJSON)
	if err != nil {
		_ = u.billingClient.ReleaseHold(ctx, req.UserID, totalAmount, "proxy_order", order.ID, "")
		_ = u.orderRepo.UpdateStatus(ctx, order.ID, domain.OrderFailed)
		return nil, fmt.Errorf("CreateOrder step5: encrypt: %w", err)
	}

	// ── Step 6: Activate order & confirm billing hold ──────────────────────
	activatedAt := time.Now()
	var expiresAt *time.Time
	if product.DurationDays != nil {
		t := activatedAt.AddDate(0, 0, *product.DurationDays)
		expiresAt = &t
	}

	if err := u.orderRepo.UpdateAfterPurchase(ctx, order.ID,
		purchaseResult.ProviderOrderID, encryptedCreds, &activatedAt, expiresAt,
	); err != nil {
		_ = u.billingClient.ReleaseHold(ctx, req.UserID, totalAmount, "proxy_order", order.ID, "")
		_ = u.orderRepo.UpdateStatus(ctx, order.ID, domain.OrderFailed)
		return nil, fmt.Errorf("CreateOrder step6: update order: %w", err)
	}

	_ = u.billingClient.ConfirmHold(ctx, req.UserID, totalAmount, "proxy_order", order.ID, fmt.Sprintf("Proxy %s", order.OrderNumber))
	// Log order.activated event
	_ = u.orderEvtRepo.Log(ctx, order.ID, domain.EventOrderActivated, map[string]any{
		"amount": order.TotalAmount.String(),
	})

	// ── Step 7: Publish fulfilled event ────────────────────────────────────
	order.Status = domain.OrderActive
	order.ActivatedAt = &activatedAt
	order.ExpiresAt = expiresAt
	order.Credentials = encryptedCreds

	go func() { _ = u.eventPub.PublishOrderFulfilled(context.Background(), order) }()

	u.logger.InfoContext(ctx, "proxy order created",
		slog.String("order_id", order.ID.String()),
		slog.String("user_id", req.UserID.String()),
		slog.String("product_id", req.ProductID.String()),
	)
	return order, nil
}

// GetCredentials returns decrypted proxy credentials for an order.
// Credentials are stored as a JSON array of ProxyCredential objects.
// Returns the first credential for single-proxy orders for backwards compat.
func (u *OrderUsecase) GetCredentials(ctx context.Context, orderID, userID uuid.UUID) (any, error) {
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("GetCredentials: %w", err)
	}
	if order.UserID != userID {
		return nil, fmt.Errorf("GetCredentials: forbidden")
	}
	if order.Status != domain.OrderActive {
		return nil, fmt.Errorf("GetCredentials: order is not active")
	}

	plain, err := u.cipher.Decrypt(order.Credentials)
	if err != nil {
		return nil, fmt.Errorf("GetCredentials: decrypt: %w", err)
	}

	// Try array first (current format: []ProxyCredential)
	var credArr []map[string]any
	if err := json.Unmarshal(plain, &credArr); err == nil {
		if len(credArr) == 1 {
			return credArr[0], nil // single proxy — return directly for frontend compat
		}
		return credArr, nil
	}

	// Fallback: legacy single-object format
	var credSingle map[string]any
	if err := json.Unmarshal(plain, &credSingle); err != nil {
		return nil, fmt.Errorf("GetCredentials: unmarshal: %w", err)
	}
	return credSingle, nil
}

// CancelOrder cancels an active order (must be in active status).
func (u *OrderUsecase) CancelOrder(ctx context.Context, orderID, userID uuid.UUID, reason string) error {
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("CancelOrder: %w", err)
	}
	if order.UserID != userID {
		return fmt.Errorf("CancelOrder: forbidden")
	}
	if order.Status != domain.OrderActive && order.Status != domain.OrderPending {
		return domain.ErrOrderNotCancellable
	}

	if err := u.orderRepo.UpdateStatus(ctx, order.ID, domain.OrderCancelled); err != nil {
		return fmt.Errorf("CancelOrder: update status: %w", err)
	}

	// Release billing hold to refund balance and create a visible transaction log
	_ = u.billingClient.ReleaseHold(ctx, order.UserID, order.TotalAmount, "proxy_order", order.ID, fmt.Sprintf("Hoàn tiền Proxy %s", order.OrderNumber))
	// Log order.cancelled event
	_ = u.orderEvtRepo.Log(ctx, order.ID, domain.EventOrderCancelled, map[string]any{
		"reason": reason,
	})

	go func() { _ = u.eventPub.PublishOrderCancelled(context.Background(), order) }()
	return nil
}

// ListOrders returns paginated orders for a user.
func (u *OrderUsecase) ListOrders(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Order, int, error) {
	return u.orderRepo.ListByUser(ctx, userID, offset, limit)
}

// GetOrder returns a single order.
func (u *OrderUsecase) GetOrder(ctx context.Context, orderID, userID uuid.UUID) (*domain.Order, error) {
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("GetOrder: %w", err)
	}
	if order.UserID != userID {
		return nil, fmt.Errorf("GetOrder: forbidden")
	}
	return order, nil
}

// RenewOrder renews an expired order during its grace period.
// New expiry is calculated from the effective expiry (COALESCE(custom_expires_at, expires_at))
// + product duration, so users do not lose days due to late renewal.
func (u *OrderUsecase) RenewOrder(ctx context.Context, orderID, userID uuid.UUID) (*domain.Order, error) {
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("RenewOrder: %w", err)
	}
	if order.UserID != userID {
		return nil, fmt.Errorf("RenewOrder: forbidden")
	}
	if order.Status != domain.OrderExpired {
		return nil, fmt.Errorf("RenewOrder: order must be in 'expired' status (current: %s)", order.Status)
	}

	// Load product to get duration and price
	product, err := u.productRepo.GetByID(ctx, order.ProductID)
	if err != nil {
		return nil, fmt.Errorf("RenewOrder: load product: %w", err)
	}
	if product.DurationDays == nil || *product.DurationDays <= 0 {
		return nil, fmt.Errorf("RenewOrder: product has no fixed duration — cannot renew")
	}

	// Calculate renewal price (same as original — use custom_price if admin set one)
	renewPrice := order.TotalAmount
	if order.CustomPrice != nil {
		renewPrice = *order.CustomPrice
	}

	// New expiry = effective_expiry + duration (NOT from now — fair to user)
	effectiveExpiry := order.ExpiresAt
	if order.CustomExpiresAt != nil {
		effectiveExpiry = order.CustomExpiresAt
	}
	if effectiveExpiry == nil {
		return nil, fmt.Errorf("RenewOrder: order has no expiry date")
	}
	newExpiry := effectiveExpiry.AddDate(0, 0, *product.DurationDays)

	// ── Billing: hold → confirm ──────────────────────────────────────────────
	if err := u.billingClient.Hold(ctx, order.UserID, renewPrice, "proxy_order", order.ID); err != nil {
		return nil, fmt.Errorf("RenewOrder: billing hold: %w", err)
	}
	if err := u.billingClient.ConfirmHold(ctx, order.UserID, renewPrice, "proxy_order", order.ID,
		fmt.Sprintf("Gia hạn Proxy %s (+%d ngày)", order.OrderNumber, *product.DurationDays),
	); err != nil {
		// Rollback hold
		_ = u.billingClient.ReleaseHold(ctx, order.UserID, renewPrice, "proxy_order", order.ID, "renew failed — release hold")
		return nil, fmt.Errorf("RenewOrder: billing confirm: %w", err)
	}

	// ── Resume at provider ───────────────────────────────────────────────────
	if order.ProviderOrderID != "" {
		provider := u.providerReg.Get(order.ProviderID.String())
		if provider != nil {
			if err := provider.Resume(ctx, order.ProviderOrderID); err != nil {
				// Log but don't fail — proxy might already be running (e.g. proxy_cheap)
				u.logger.WarnContext(ctx, "RenewOrder: resume at provider failed (continuing)", slog.String("err", err.Error()))
			}
		}
	}

	// ── Update order in DB ───────────────────────────────────────────────────
	// Clear custom_expires_at and set expires_at = new effective expiry
	if err := u.orderRepo.UpdateAfterPurchase(ctx, order.ID, order.ProviderOrderID, order.Credentials, order.ActivatedAt, &newExpiry); err != nil {
		return nil, fmt.Errorf("RenewOrder: update: %w", err)
	}
	// Clear custom_expires_at so newExpiry becomes the canonical expiry
	if err := u.orderRepo.UpdateOrder(ctx, order.ID, nil, nil, order.AdminNote); err != nil {
		u.logger.WarnContext(ctx, "RenewOrder: clear custom_expires_at", slog.String("err", err.Error()))
	}
	if err := u.orderRepo.UpdateStatus(ctx, order.ID, domain.OrderActive); err != nil {
		return nil, fmt.Errorf("RenewOrder: update status: %w", err)
	}

	// Log renewal event
	_ = u.orderEvtRepo.Log(ctx, order.ID, domain.EventOrderRenewed, map[string]any{
		"new_expires_at": newExpiry.Format(time.RFC3339),
		"duration_days":  *product.DurationDays,
		"amount":         renewPrice.String(),
	})

	u.logger.InfoContext(ctx, "proxy renewed",
		slog.String("order_id", order.ID.String()),
		slog.String("new_expires_at", newExpiry.Format(time.RFC3339)),
	)

	order.Status = domain.OrderActive
	order.ExpiresAt = &newExpiry
	order.CustomExpiresAt = nil
	return order, nil
}
