// Package usecase contains all business logic for billing-service.
// wallet_usecase.go handles: Deduct, Hold, ConfirmHold, ReleaseHold, GetBalance.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/pvp/billing-service/internal/domain"
	"github.com/shopspring/decimal"
)

const lowBalanceThreshold = 5.00

// WalletUsecase handles all wallet balance operations.
// All mutations run inside DB transactions with SELECT FOR UPDATE.
type WalletUsecase struct {
	walletRepo domain.IWalletRepository
	txnRepo    domain.ITransactionRepository
	txRunner   domain.ITxRunner
	eventPub   domain.IEventPublisher
	logger     *slog.Logger
}

// NewWalletUsecase constructs WalletUsecase via DI.
func NewWalletUsecase(
	walletRepo domain.IWalletRepository,
	txnRepo domain.ITransactionRepository,
	txRunner domain.ITxRunner,
	eventPub domain.IEventPublisher,
	logger *slog.Logger,
) *WalletUsecase {
	return &WalletUsecase{
		walletRepo: walletRepo,
		txnRepo:    txnRepo,
		txRunner:   txRunner,
		eventPub:   eventPub,
		logger:     logger,
	}
}

// CreateWallet creates a new wallet for a user.
func (u *WalletUsecase) CreateWallet(ctx context.Context, userID uuid.UUID) (*domain.Wallet, error) {
	w := &domain.Wallet{
		ID:                  uuid.New(),
		UserID:              userID,
		Balance:             decimal.Zero,
		HoldAmount:          decimal.Zero,
		Currency:            "USD",
		LowBalanceThreshold: decimal.NewFromFloat(lowBalanceThreshold),
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	if err := u.walletRepo.Create(ctx, w); err != nil {
		return nil, fmt.Errorf("WalletUsecase.CreateWallet: %w", err)
	}
	return w, nil
}

// GetBalance returns the wallet for a user.
func (u *WalletUsecase) GetBalance(ctx context.Context, userID uuid.UUID) (*domain.Wallet, error) {
	w, err := u.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("WalletUsecase.GetBalance: %w", err)
	}
	return w, nil
}

// DeductRequest is the input for Deduct.
type DeductRequest struct {
	UserID        uuid.UUID
	Amount        decimal.Decimal
	ReferenceType string
	ReferenceID   *uuid.UUID
	Description   string
	RequestID     *uuid.UUID
}

// Deduct immediately deducts amount from wallet balance.
// Uses SELECT FOR UPDATE to prevent race conditions.
func (u *WalletUsecase) Deduct(ctx context.Context, req DeductRequest) (*domain.Transaction, error) {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrInvalidAmount
	}

	var txn *domain.Transaction

	err := u.txRunner.RunInTx(ctx, func(ctx context.Context) error {
		// Lock wallet row
		w, err := u.walletRepo.GetByUserIDForUpdate(ctx, req.UserID)
		if err != nil {
			return fmt.Errorf("Deduct: get wallet: %w", err)
		}

		if w.AvailableBalance().LessThan(req.Amount) {
			return domain.ErrInsufficientFunds
		}

		newBalance := w.Balance.Sub(req.Amount)
		if err := u.walletRepo.UpdateBalance(ctx, w.ID, newBalance, w.HoldAmount); err != nil {
			return fmt.Errorf("Deduct: update wallet: %w", err)
		}

		txn = &domain.Transaction{
			ID:            uuid.New(),
			TxnNumber:     generateTxnNumber(),
			WalletID:      w.ID,
			UserID:        req.UserID,
			Type:          domain.TxnOrderCharge,
			Amount:        req.Amount.Neg(),
			BalanceBefore: w.Balance,
			BalanceAfter:  newBalance,
			ReferenceType: req.ReferenceType,
			ReferenceID:   req.ReferenceID,
			Description:   req.Description,
			RequestID:     req.RequestID,
			CreatedAt:     time.Now(),
		}
		if err := u.txnRepo.Create(ctx, txn); err != nil {
			return fmt.Errorf("Deduct: create txn: %w", err)
		}

		// Check low balance
		if newBalance.LessThanOrEqual(decimal.NewFromFloat(lowBalanceThreshold)) {
			if newBalance.IsZero() {
				go func() { _ = u.eventPub.PublishWalletEmpty(context.Background(), req.UserID) }()
			} else {
				go func() { _ = u.eventPub.PublishWalletLow(context.Background(), req.UserID, newBalance) }()
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	go func() {
		_ = u.eventPub.PublishCharged(context.Background(), req.UserID, req.Amount, req.ReferenceType)
	}()

	u.logger.InfoContext(ctx, "wallet deducted",
		slog.String("user_id", req.UserID.String()),
		slog.String("amount", req.Amount.String()),
		slog.String("txn", txn.TxnNumber),
	)
	return txn, nil
}

// HoldRequest is the input for Hold.
type HoldRequest struct {
	UserID        uuid.UUID
	Amount        decimal.Decimal
	ReferenceType string
	ReferenceID   *uuid.UUID
	Description   string
}

// Hold reserves amount in hold_amount (does not reduce balance immediately).
func (u *WalletUsecase) Hold(ctx context.Context, req HoldRequest) (*domain.Transaction, error) {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrInvalidAmount
	}

	var txn *domain.Transaction
	err := u.txRunner.RunInTx(ctx, func(ctx context.Context) error {
		w, err := u.walletRepo.GetByUserIDForUpdate(ctx, req.UserID)
		if err != nil {
			return fmt.Errorf("Hold: get wallet: %w", err)
		}
		if w.AvailableBalance().LessThan(req.Amount) {
			return domain.ErrInsufficientFunds
		}

		newHold := w.HoldAmount.Add(req.Amount)
		if err := u.walletRepo.UpdateBalance(ctx, w.ID, w.Balance, newHold); err != nil {
			return fmt.Errorf("Hold: update: %w", err)
		}

		txn = &domain.Transaction{
			ID:            uuid.New(),
			TxnNumber:     generateTxnNumber(),
			WalletID:      w.ID,
			UserID:        req.UserID,
			Type:          domain.TxnHold,
			Amount:        req.Amount,
			BalanceBefore: w.Balance,
			BalanceAfter:  w.Balance,
			ReferenceType: req.ReferenceType,
			ReferenceID:   req.ReferenceID,
			Description:   req.Description,
			CreatedAt:     time.Now(),
		}
		return u.txnRepo.Create(ctx, txn)
	})
	if err != nil {
		return nil, err
	}
	return txn, nil
}

// ConfirmHold transfers hold_amount to actual deduction (reduces both balance and hold).
func (u *WalletUsecase) ConfirmHold(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, refType string, refID *uuid.UUID) (*domain.Transaction, error) {
	var txn *domain.Transaction
	err := u.txRunner.RunInTx(ctx, func(ctx context.Context) error {
		w, err := u.walletRepo.GetByUserIDForUpdate(ctx, userID)
		if err != nil {
			return fmt.Errorf("ConfirmHold: get wallet: %w", err)
		}
		if w.HoldAmount.LessThan(amount) {
			return fmt.Errorf("ConfirmHold: hold_amount %s < amount %s", w.HoldAmount, amount)
		}

		newBalance := w.Balance.Sub(amount)
		newHold := w.HoldAmount.Sub(amount)
		if err := u.walletRepo.UpdateBalance(ctx, w.ID, newBalance, newHold); err != nil {
			return fmt.Errorf("ConfirmHold: update: %w", err)
		}

		txn = &domain.Transaction{
			ID:            uuid.New(),
			TxnNumber:     generateTxnNumber(),
			WalletID:      w.ID,
			UserID:        userID,
			Type:          domain.TxnHoldConfirm,
			Amount:        amount.Neg(),
			BalanceBefore: w.Balance,
			BalanceAfter:  newBalance,
			ReferenceType: refType,
			ReferenceID:   refID,
			CreatedAt:     time.Now(),
		}
		return u.txnRepo.Create(ctx, txn)
	})
	if err != nil {
		return nil, err
	}
	return txn, nil
}

// ReleaseHold releases a previously held amount back to available.
func (u *WalletUsecase) ReleaseHold(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, refType string, refID *uuid.UUID) (*domain.Transaction, error) {
	var txn *domain.Transaction
	err := u.txRunner.RunInTx(ctx, func(ctx context.Context) error {
		w, err := u.walletRepo.GetByUserIDForUpdate(ctx, userID)
		if err != nil {
			return fmt.Errorf("ReleaseHold: get wallet: %w", err)
		}

		newHold := w.HoldAmount.Sub(amount)
		if newHold.IsNegative() {
			newHold = decimal.Zero
		}
		if err := u.walletRepo.UpdateBalance(ctx, w.ID, w.Balance, newHold); err != nil {
			return fmt.Errorf("ReleaseHold: update: %w", err)
		}

		txn = &domain.Transaction{
			ID:            uuid.New(),
			TxnNumber:     generateTxnNumber(),
			WalletID:      w.ID,
			UserID:        userID,
			Type:          domain.TxnHoldRelease,
			Amount:        amount,
			BalanceBefore: w.Balance,
			BalanceAfter:  w.Balance,
			ReferenceType: refType,
			ReferenceID:   refID,
			CreatedAt:     time.Now(),
		}
		return u.txnRepo.Create(ctx, txn)
	})
	if err != nil {
		return nil, err
	}
	return txn, nil
}

// Credit adds funds to a wallet (after successful deposit).
func (u *WalletUsecase) Credit(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, refType string, refID *uuid.UUID, description string) (*domain.Transaction, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrInvalidAmount
	}

	var txn *domain.Transaction
	err := u.txRunner.RunInTx(ctx, func(ctx context.Context) error {
		w, err := u.walletRepo.GetByUserIDForUpdate(ctx, userID)
		if err != nil {
			if !errors.Is(err, domain.ErrWalletNotFound) {
				return fmt.Errorf("Credit: get wallet: %w", err)
			}
			// Auto-create wallet for users who haven't accessed billing yet.
			newWallet := &domain.Wallet{
				ID:                  uuid.New(),
				UserID:              userID,
				Balance:             decimal.Zero,
				HoldAmount:          decimal.Zero,
				Currency:            "USD",
				LowBalanceThreshold: decimal.NewFromFloat(lowBalanceThreshold),
				CreatedAt:           time.Now(),
				UpdatedAt:           time.Now(),
			}
			if createErr := u.walletRepo.Create(ctx, newWallet); createErr != nil {
				return fmt.Errorf("Credit: create wallet: %w", createErr)
			}
			w = newWallet
		}

		newBalance := w.Balance.Add(amount)
		if err := u.walletRepo.UpdateBalance(ctx, w.ID, newBalance, w.HoldAmount); err != nil {
			return fmt.Errorf("Credit: update: %w", err)
		}

		txn = &domain.Transaction{
			ID:            uuid.New(),
			TxnNumber:     generateTxnNumber(),
			WalletID:      w.ID,
			UserID:        userID,
			Type:          domain.TxnDeposit,
			Amount:        amount,
			BalanceBefore: w.Balance,
			BalanceAfter:  newBalance,
			ReferenceType: refType,
			ReferenceID:   refID,
			Description:   description,
			CreatedAt:     time.Now(),
		}
		return u.txnRepo.Create(ctx, txn)
	})
	if err != nil {
		return nil, err
	}

	go func() { _ = u.eventPub.PublishDepositCompleted(context.Background(), userID, amount) }()
	return txn, nil
}

// ListTransactions returns paginated transactions for a user.
func (u *WalletUsecase) ListTransactions(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Transaction, int, error) {
	return u.txnRepo.ListByUser(ctx, userID, offset, limit)
}

// generateTxnNumber creates a unique transaction number.
// Production: use a DB sequence e.g. nextval('billing.txn_number_seq').
func generateTxnNumber() string {
	return fmt.Sprintf("TXN-%d", time.Now().UnixNano())
}
