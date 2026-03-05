// Package usecase contains reseller-service business logic.
package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/pvp/reseller-service/internal/domain"
	"github.com/shopspring/decimal"
)

// ─── Reseller Admin Usecase ───────────────────────────────────────────────────

// ResellerUsecase handles reseller account management by admins and resellers themselves.
type ResellerUsecase struct {
	resellerRepo  domain.IResellerRepository
	pricingRepo   domain.IPricingRepository
	subAccountRepo domain.ISubAccountRepository
	eventPub      domain.IEventPublisher
	logger        *slog.Logger
}

func NewResellerUsecase(
	resellerRepo domain.IResellerRepository,
	pricingRepo domain.IPricingRepository,
	subAccountRepo domain.ISubAccountRepository,
	eventPub domain.IEventPublisher,
	logger *slog.Logger,
) *ResellerUsecase {
	return &ResellerUsecase{
		resellerRepo:  resellerRepo,
		pricingRepo:   pricingRepo,
		subAccountRepo: subAccountRepo,
		eventPub:      eventPub,
		logger:        logger,
	}
}

// CreateResellerRequest is the input for creating a reseller account.
type CreateResellerRequest struct {
	UserID        uuid.UUID
	CompanyName   string
	Email         string
	Phone         string
	Address       string
	TaxID         string
	CommissionPct decimal.Decimal
}

// CreateReseller creates a pending reseller account (admin-initiated or self-service).
func (u *ResellerUsecase) CreateReseller(ctx context.Context, req CreateResellerRequest) (*domain.ResellerAccount, error) {
	// Check for existing account
	if existing, err := u.resellerRepo.GetByUserID(ctx, req.UserID); err == nil && existing != nil {
		return nil, domain.ErrResellerAlreadyExists
	}

	reseller := &domain.ResellerAccount{
		ID:            uuid.New(),
		UserID:        req.UserID,
		CompanyName:   req.CompanyName,
		Email:         req.Email,
		Phone:         req.Phone,
		Address:       req.Address,
		TaxID:         req.TaxID,
		Status:        domain.StatusPending,
		CommissionPct: req.CommissionPct,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := u.resellerRepo.Create(ctx, reseller); err != nil {
		return nil, fmt.Errorf("CreateReseller: %w", err)
	}

	_ = u.eventPub.PublishCreated(ctx, reseller)
	u.logger.InfoContext(ctx, "reseller account created",
		slog.String("reseller_id", reseller.ID.String()),
		slog.String("company", req.CompanyName),
	)
	return reseller, nil
}

// ApproveReseller approves a pending reseller (admin only).
func (u *ResellerUsecase) ApproveReseller(ctx context.Context, resellerID uuid.UUID) (*domain.ResellerAccount, error) {
	reseller, err := u.resellerRepo.GetByID(ctx, resellerID)
	if err != nil {
		return nil, fmt.Errorf("ApproveReseller: %w", err)
	}
	if reseller.Status == domain.StatusApproved {
		return reseller, nil // idempotent
	}

	now := time.Now()
	if err := u.resellerRepo.UpdateStatus(ctx, resellerID, domain.StatusApproved, &now, nil, ""); err != nil {
		return nil, fmt.Errorf("ApproveReseller: update status: %w", err)
	}

	_ = u.eventPub.PublishApproved(ctx, resellerID)
	u.logger.InfoContext(ctx, "reseller approved", slog.String("reseller_id", resellerID.String()))
	reseller.Status = domain.StatusApproved
	reseller.ApprovedAt = &now
	return reseller, nil
}

// SuspendReseller suspends an active reseller (admin only).
func (u *ResellerUsecase) SuspendReseller(ctx context.Context, resellerID uuid.UUID, reason string) error {
	reseller, err := u.resellerRepo.GetByID(ctx, resellerID)
	if err != nil {
		return fmt.Errorf("SuspendReseller: %w", err)
	}
	if reseller.Status == domain.StatusSuspended {
		return nil // idempotent
	}

	now := time.Now()
	if err := u.resellerRepo.UpdateStatus(ctx, resellerID, domain.StatusSuspended, nil, &now, reason); err != nil {
		return fmt.Errorf("SuspendReseller: %w", err)
	}

	_ = u.eventPub.PublishSuspended(ctx, resellerID, reason)
	u.logger.InfoContext(ctx, "reseller suspended",
		slog.String("reseller_id", resellerID.String()),
		slog.String("reason", reason),
	)
	return nil
}

// GetReseller returns a reseller by ID.
func (u *ResellerUsecase) GetReseller(ctx context.Context, id uuid.UUID) (*domain.ResellerAccount, error) {
	return u.resellerRepo.GetByID(ctx, id)
}

// GetResellerByUserID resolves the reseller account for a user.
func (u *ResellerUsecase) GetResellerByUserID(ctx context.Context, userID uuid.UUID) (*domain.ResellerAccount, error) {
	return u.resellerRepo.GetByUserID(ctx, userID)
}

// ListResellers returns paginated resellers (admin).
func (u *ResellerUsecase) ListResellers(ctx context.Context, status string, offset, limit int) ([]*domain.ResellerAccount, int, error) {
	return u.resellerRepo.List(ctx, status, offset, limit)
}

// ─── Pricing Usecase ──────────────────────────────────────────────────────────

// SetPricingRequest is the input for a reseller setting a product sell price.
type SetPricingRequest struct {
	ResellerID  uuid.UUID
	ProductID   uuid.UUID
	ProductType string
	CostPrice   decimal.Decimal // provided by admin; reseller cannot override
	FloorPrice  decimal.Decimal // minimum allowed sell price
	SellPrice   decimal.Decimal // reseller's chosen sell price
}

// SetPricing sets or updates a reseller's sell price for a product.
// Validation: sell_price >= floor_price >= cost_price; sell_price <= cost_price * 100
func (u *ResellerUsecase) SetPricing(ctx context.Context, req SetPricingRequest) (*domain.PricingOverride, error) {
	// Pricing validation per spec
	if req.SellPrice.LessThan(req.CostPrice) {
		return nil, domain.ErrPriceBelowCost
	}
	if req.FloorPrice.IsPositive() && req.SellPrice.LessThan(req.FloorPrice) {
		return nil, domain.ErrPriceBelowFloor
	}
	cap := req.CostPrice.Mul(decimal.NewFromInt(100))
	if req.SellPrice.GreaterThan(cap) {
		return nil, domain.ErrPriceAboveCap
	}

	pricing := &domain.PricingOverride{
		ID:          uuid.New(),
		ResellerID:  req.ResellerID,
		ProductID:   req.ProductID,
		ProductType: req.ProductType,
		CostPrice:   req.CostPrice,
		FloorPrice:  req.FloorPrice,
		SellPrice:   req.SellPrice,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := u.pricingRepo.Upsert(ctx, pricing); err != nil {
		return nil, fmt.Errorf("SetPricing: %w", err)
	}

	_ = u.eventPub.PublishPricingUpdated(ctx, req.ResellerID, req.ProductID, req.SellPrice)
	u.logger.InfoContext(ctx, "reseller pricing updated",
		slog.String("reseller_id", req.ResellerID.String()),
		slog.String("product_id", req.ProductID.String()),
		slog.String("sell_price", req.SellPrice.String()),
	)
	return pricing, nil
}

// ListPricing returns all pricing overrides for a reseller.
func (u *ResellerUsecase) ListPricing(ctx context.Context, resellerID uuid.UUID) ([]*domain.PricingOverride, error) {
	return u.pricingRepo.ListByReseller(ctx, resellerID)
}

// ─── Sub-account Usecase ──────────────────────────────────────────────────────

// CreateSubAccountRequest creates a sub-account under a reseller.
type CreateSubAccountRequest struct {
	ResellerID  uuid.UUID
	UserID      uuid.UUID
	CreditLimit decimal.Decimal
}

func (u *ResellerUsecase) CreateSubAccount(ctx context.Context, req CreateSubAccountRequest) (*domain.SubAccount, error) {
	// Verify reseller is approved
	reseller, err := u.resellerRepo.GetByID(ctx, req.ResellerID)
	if err != nil {
		return nil, fmt.Errorf("CreateSubAccount: %w", err)
	}
	if reseller.Status != domain.StatusApproved {
		return nil, domain.ErrResellerNotApproved
	}

	sub := &domain.SubAccount{
		ID:          uuid.New(),
		ResellerID:  req.ResellerID,
		UserID:      req.UserID,
		CreditLimit: req.CreditLimit,
		CreatedAt:   time.Now(),
	}
	if err := u.subAccountRepo.Create(ctx, sub); err != nil {
		return nil, fmt.Errorf("CreateSubAccount: %w", err)
	}
	return sub, nil
}

func (u *ResellerUsecase) ListSubAccounts(ctx context.Context, resellerID uuid.UUID, offset, limit int) ([]*domain.SubAccount, int, error) {
	return u.subAccountRepo.ListByReseller(ctx, resellerID, offset, limit)
}

func (u *ResellerUsecase) SetCreditLimit(ctx context.Context, subID uuid.UUID, resellerID uuid.UUID, limit decimal.Decimal) error {
	sub, err := u.subAccountRepo.GetByUserID(ctx, subID)
	if err != nil {
		return domain.ErrSubAccountNotFound
	}
	if sub.ResellerID != resellerID {
		return domain.ErrForbidden
	}
	return u.subAccountRepo.UpdateCreditLimit(ctx, sub.ID, limit)
}

// ─── API Key Usecase ──────────────────────────────────────────────────────────

// APIKeyUsecase manages reseller API keys.
type APIKeyUsecase struct {
	apiKeyRepo domain.IAPIKeyRepository
	logger     *slog.Logger
}

func NewAPIKeyUsecase(apiKeyRepo domain.IAPIKeyRepository, logger *slog.Logger) *APIKeyUsecase {
	return &APIKeyUsecase{apiKeyRepo: apiKeyRepo, logger: logger}
}

// CreateAPIKeyResult is returned when an API key is created; plaintext shown once.
type CreateAPIKeyResult struct {
	APIKey     *domain.ResellerAPIKey
	PlainKey   string // shown once to user, never stored
}

// CreateAPIKey generates a cryptographically random reseller API key.
func (u *APIKeyUsecase) CreateAPIKey(ctx context.Context, resellerID uuid.UUID, name string, scopes []string, expiresAt *time.Time) (*CreateAPIKeyResult, error) {
	// Generate 32-byte random key
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return nil, fmt.Errorf("CreateAPIKey: generate: %w", err)
	}
	// Format: rvk_<hex> (reseller api key prefix)
	plainKey := "rvk_" + hex.EncodeToString(rawBytes)

	// Hash for storage
	h := sha256.Sum256([]byte(plainKey))
	keyHash := hex.EncodeToString(h[:])

	apiKey := &domain.ResellerAPIKey{
		ID:         uuid.New(),
		ResellerID: resellerID,
		Name:       name,
		KeyHash:    keyHash,
		KeyPrefix:  plainKey[:12], // "rvk_" + first 8 hex chars
		Scopes:     scopes,
		ExpiresAt:  expiresAt,
		CreatedAt:  time.Now(),
	}

	if err := u.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("CreateAPIKey: save: %w", err)
	}

	u.logger.InfoContext(ctx, "reseller API key created",
		slog.String("reseller_id", resellerID.String()),
		slog.String("name", name),
		slog.String("prefix", apiKey.KeyPrefix),
	)
	return &CreateAPIKeyResult{APIKey: apiKey, PlainKey: plainKey}, nil
}

// ValidateAPIKey validates a raw API key and returns the DB record.
func (u *APIKeyUsecase) ValidateAPIKey(ctx context.Context, rawKey string) (*domain.ResellerAPIKey, error) {
	h := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(h[:])

	k, err := u.apiKeyRepo.GetByHash(ctx, keyHash)
	if err != nil {
		return nil, domain.ErrAPIKeyNotFound
	}
	if k.RevokedAt != nil {
		return nil, domain.ErrAPIKeyRevoked
	}
	if k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt) {
		return nil, domain.ErrAPIKeyExpired
	}

	// Fire-and-forget update last_used
	go func() { _ = u.apiKeyRepo.UpdateLastUsed(context.Background(), k.ID) }()
	return k, nil
}

// ListAPIKeys returns all API keys for a reseller.
func (u *APIKeyUsecase) ListAPIKeys(ctx context.Context, resellerID uuid.UUID) ([]*domain.ResellerAPIKey, error) {
	return u.apiKeyRepo.ListByReseller(ctx, resellerID)
}

// RevokeAPIKey revokes an API key.
func (u *APIKeyUsecase) RevokeAPIKey(ctx context.Context, keyID, resellerID uuid.UUID) error {
	keys, err := u.apiKeyRepo.ListByReseller(ctx, resellerID)
	if err != nil {
		return fmt.Errorf("RevokeAPIKey: list: %w", err)
	}
	for _, k := range keys {
		if k.ID == keyID {
			return u.apiKeyRepo.Revoke(ctx, keyID)
		}
	}
	return domain.ErrAPIKeyNotFound
}
