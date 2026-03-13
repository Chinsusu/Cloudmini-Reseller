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
	ConfirmHold(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, refType string, refID uuid.UUID) error
	ReleaseHold(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, refType string, refID uuid.UUID) error
	CalculatePrice(ctx context.Context, baseCost decimal.Decimal, productType string, productID uuid.UUID, resellerID *uuid.UUID) (decimal.Decimal, error)
}

// OrderUsecase handles the proxy order lifecycle.
type OrderUsecase struct {
	orderRepo    domain.IOrderRepository
	productRepo  domain.IProductRepository
	providerRepo domain.IProviderRepository
	providerReg  *providers.Registry
	billingClient BillingClient
	cipher       *cryptopkg.Cipher
	eventPub     domain.IEventPublisher
	logger       *slog.Logger
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
		OrderNumber:    fmt.Sprintf("PX-%d", time.Now().UnixNano()),
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
		_ = u.billingClient.ReleaseHold(ctx, req.UserID, totalAmount, "proxy_order", order.ID)
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
		_ = u.billingClient.ReleaseHold(ctx, req.UserID, totalAmount, "proxy_order", order.ID)
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
			_ = u.billingClient.ReleaseHold(ctx, req.UserID, totalAmount, "proxy_order", order.ID)
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
		_ = u.billingClient.ReleaseHold(ctx, req.UserID, totalAmount, "proxy_order", order.ID)
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
		_ = u.billingClient.ReleaseHold(ctx, req.UserID, totalAmount, "proxy_order", order.ID)
		_ = u.orderRepo.UpdateStatus(ctx, order.ID, domain.OrderFailed)
		return nil, fmt.Errorf("CreateOrder step6: update order: %w", err)
	}

	_ = u.billingClient.ConfirmHold(ctx, req.UserID, totalAmount, "proxy_order", order.ID)

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
func (u *OrderUsecase) GetCredentials(ctx context.Context, orderID, userID uuid.UUID) (map[string]any, error) {
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

	var creds map[string]any
	if err := json.Unmarshal(plain, &creds); err != nil {
		return nil, fmt.Errorf("GetCredentials: unmarshal: %w", err)
	}
	return creds, nil
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
