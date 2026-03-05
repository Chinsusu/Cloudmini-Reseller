package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/pvp/billing-service/internal/domain"
	"github.com/shopspring/decimal"
)

// PricingEngine calculates sell price from base cost + markup rules.
type PricingEngine struct {
	pricingRepo domain.IPricingRepository
	logger      *slog.Logger
}

// NewPricingEngine constructs PricingEngine.
func NewPricingEngine(pricingRepo domain.IPricingRepository, logger *slog.Logger) *PricingEngine {
	return &PricingEngine{pricingRepo: pricingRepo, logger: logger}
}

// CalculatePrice determines the price for a product for a given user/reseller.
// Priority: reseller-specific rule > global product rule > global type rule > base cost.
func (e *PricingEngine) CalculatePrice(ctx context.Context, baseCost decimal.Decimal, productType string, productID uuid.UUID, resellerID *uuid.UUID) (decimal.Decimal, error) {
	// 1. Try reseller-specific rule for this product
	if resellerID != nil {
		rule, err := e.pricingRepo.GetRule(ctx, resellerID, productType, &productID)
		if err == nil {
			return e.applyRule(baseCost, rule), nil
		}
		// 2. Try reseller-level type rule
		rule, err = e.pricingRepo.GetRule(ctx, resellerID, productType, nil)
		if err == nil {
			return e.applyRule(baseCost, rule), nil
		}
	}

	// 3. Try global product rule
	rule, err := e.pricingRepo.GetRule(ctx, nil, productType, &productID)
	if err == nil {
		return e.applyRule(baseCost, rule), nil
	}

	// 4. Try global type rule
	rule, err = e.pricingRepo.GetRule(ctx, nil, productType, nil)
	if err == nil {
		return e.applyRule(baseCost, rule), nil
	}

	// 5. No rule — return base cost as-is
	return baseCost, nil
}

func (e *PricingEngine) applyRule(base decimal.Decimal, rule *domain.PricingRule) decimal.Decimal {
	var price decimal.Decimal
	switch rule.MarkupType {
	case "percentage":
		// price = base * (1 + markup_value/100)
		multiplier := decimal.NewFromInt(1).Add(rule.MarkupValue.Div(decimal.NewFromInt(100)))
		price = base.Mul(multiplier)
	case "fixed":
		price = base.Add(rule.MarkupValue)
	default:
		price = base
	}

	// Apply minimum price floor
	if rule.MinPrice != nil && price.LessThan(*rule.MinPrice) {
		price = *rule.MinPrice
	}

	return price.Round(4)
}

// ─── PaymentUsecase ───────────────────────────────────────────────────────────

// PaymentUsecase handles deposit creation and webhook processing.
type PaymentUsecase struct {
	paymentRepo domain.IPaymentRepository
	walletUC    *WalletUsecase
	logger      *slog.Logger

	stripeSecretKey string
	frontendBaseURL string
}

// NewPaymentUsecase constructs PaymentUsecase.
func NewPaymentUsecase(
	paymentRepo domain.IPaymentRepository,
	walletUC *WalletUsecase,
	logger *slog.Logger,
	stripeSecretKey, frontendBaseURL string,
) *PaymentUsecase {
	return &PaymentUsecase{
		paymentRepo:     paymentRepo,
		walletUC:        walletUC,
		logger:          logger,
		stripeSecretKey: stripeSecretKey,
		frontendBaseURL: frontendBaseURL,
	}
}

// CreateDepositRequest is the input for creating a deposit link.
type CreateDepositRequest struct {
	UserID   uuid.UUID
	WalletID uuid.UUID
	Gateway  string
	Amount   decimal.Decimal
	Currency string
}

// CreateDepositResult contains the checkout URL.
type CreateDepositResult struct {
	Payment    *domain.Payment
	CheckoutURL string
}

// CreateDeposit creates a pending payment and returns a gateway checkout URL.
func (u *PaymentUsecase) CreateDeposit(ctx context.Context, req CreateDepositRequest) (*CreateDepositResult, error) {
	if req.Amount.LessThan(decimal.NewFromFloat(1.0)) {
		return nil, fmt.Errorf("PaymentUsecase.CreateDeposit: minimum deposit is 1.00")
	}

	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	payment := &domain.Payment{
		ID:            uuid.New(),
		PaymentNumber: generatePaymentNumber(),
		UserID:        req.UserID,
		WalletID:      req.WalletID,
		Gateway:       req.Gateway,
		Amount:        req.Amount,
		Currency:      currency,
		Status:        domain.PaymentPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := u.paymentRepo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("PaymentUsecase.CreateDeposit: save: %w", err)
	}

	// Build checkout URL based on gateway
	checkoutURL, err := u.buildCheckoutURL(ctx, payment)
	if err != nil {
		return nil, fmt.Errorf("PaymentUsecase.CreateDeposit: build checkout: %w", err)
	}

	u.logger.InfoContext(ctx, "deposit created",
		slog.String("user_id", req.UserID.String()),
		slog.String("payment_id", payment.ID.String()),
		slog.String("gateway", req.Gateway),
		slog.String("amount", req.Amount.String()),
	)

	return &CreateDepositResult{Payment: payment, CheckoutURL: checkoutURL}, nil
}

// HandleWebhook processes a confirmed payment from a gateway and credits the wallet.
func (u *PaymentUsecase) HandleWebhook(ctx context.Context, gateway, gatewayTxnID string, amount decimal.Decimal) error {
	// Idempotency: check if already processed
	existing, err := u.paymentRepo.GetByGatewayTxnID(ctx, gatewayTxnID)
	if err == nil && existing.Status == domain.PaymentCompleted {
		u.logger.WarnContext(ctx, "duplicate webhook received", slog.String("gateway_txn_id", gatewayTxnID))
		return nil
	}
	if err != nil {
		return fmt.Errorf("HandleWebhook: get payment: %w", err)
	}

	// Update payment status
	now := time.Now()
	existing.Status = domain.PaymentCompleted
	existing.GatewayTxnID = gatewayTxnID
	existing.CompletedAt = &now
	if err := u.paymentRepo.UpdateStatus(ctx, existing.ID, domain.PaymentCompleted, gatewayTxnID); err != nil {
		return fmt.Errorf("HandleWebhook: update status: %w", err)
	}

	// Credit wallet
	ref := existing.ID
	_, err = u.walletUC.Credit(ctx, existing.UserID, existing.Amount, "payment", &ref,
		fmt.Sprintf("Deposit via %s — %s", gateway, existing.PaymentNumber))
	if err != nil {
		return fmt.Errorf("HandleWebhook: credit wallet: %w", err)
	}

	u.logger.InfoContext(ctx, "deposit completed",
		slog.String("user_id", existing.UserID.String()),
		slog.String("payment_id", existing.ID.String()),
		slog.String("amount", existing.Amount.String()),
	)
	return nil
}

func (u *PaymentUsecase) buildCheckoutURL(_ context.Context, p *domain.Payment) (string, error) {
	switch p.Gateway {
	case "stripe":
		// Production: call stripe.CheckoutSessions.New()
		// For now return a demo URL
		return fmt.Sprintf("%s/billing/pending?payment_id=%s&gateway=stripe", u.frontendBaseURL, p.ID), nil
	case "vnpay":
		return fmt.Sprintf("%s/billing/pending?payment_id=%s&gateway=vnpay", u.frontendBaseURL, p.ID), nil
	case "momo":
		return fmt.Sprintf("%s/billing/pending?payment_id=%s&gateway=momo", u.frontendBaseURL, p.ID), nil
	default:
		return "", domain.ErrUnsupportedGateway
	}
}

func generatePaymentNumber() string {
	return fmt.Sprintf("PAY-%d", time.Now().UnixNano())
}
