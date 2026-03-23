package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/pvp/proxy-service/internal/domain"
	"github.com/pvp/proxy-service/internal/providers"
)

// LockOrderRequest is the input for LockOrder / UnlockOrder.
type LockOrderRequest struct {
	OrderID uuid.UUID
	// Reason is stored in the order event payload (auditing).
	Reason string
}

// LockUsecase handles admin-initiated lock/unlock of proxy orders.
// Lock  → provider.Suspend (PUT action=lock) + status "suspended"
// Unlock → provider.Resume  (PUT action=unlock) + status "active"
type LockUsecase struct {
	orderRepo    domain.IOrderRepository
	orderEvtRepo domain.IOrderEventRepository
	providerReg  *providers.Registry
	logger       *slog.Logger
}

// NewLockUsecase creates a LockUsecase.
func NewLockUsecase(
	orderRepo domain.IOrderRepository,
	orderEvtRepo domain.IOrderEventRepository,
	providerReg *providers.Registry,
	logger *slog.Logger,
) *LockUsecase {
	return &LockUsecase{
		orderRepo:    orderRepo,
		orderEvtRepo: orderEvtRepo,
		providerReg:  providerReg,
		logger:       logger,
	}
}

// LockOrder suspends a proxy at the provider and sets its status to "suspended".
// Only active proxies can be locked. Admin-only action.
func (u *LockUsecase) LockOrder(ctx context.Context, req LockOrderRequest) error {
	order, err := u.orderRepo.GetByID(ctx, req.OrderID)
	if err != nil {
		return fmt.Errorf("LockOrder: %w", err)
	}
	if order.Status != domain.OrderActive && order.Status != domain.OrderExpired {
		return fmt.Errorf("LockOrder: order status must be active or expired (current: %s)", order.Status)
	}

	// Suspend at provider (best-effort)
	if order.ProviderOrderID != "" {
		provider := u.providerReg.Get(order.ProviderID.String())
		if provider != nil {
			if err := provider.Suspend(ctx, order.ProviderOrderID); err != nil {
				u.logger.WarnContext(ctx, "LockOrder: suspend at provider failed (continuing)", slog.String("err", err.Error()))
			}
		}
	}

	if err := u.orderRepo.UpdateStatus(ctx, req.OrderID, domain.OrderSuspended); err != nil {
		return fmt.Errorf("LockOrder: update status: %w", err)
	}
	_ = u.orderEvtRepo.Log(ctx, req.OrderID, domain.EventOrderLocked, map[string]any{
		"reason": req.Reason,
	})
	u.logger.InfoContext(ctx, "proxy locked by admin",
		slog.String("order_id", req.OrderID.String()),
		slog.String("reason", req.Reason),
	)
	return nil
}

// UnlockOrder resumes a suspended proxy at the provider and sets its status back to "active".
// Only suspended proxies can be unlocked. Admin-only action.
func (u *LockUsecase) UnlockOrder(ctx context.Context, req LockOrderRequest) error {
	order, err := u.orderRepo.GetByID(ctx, req.OrderID)
	if err != nil {
		return fmt.Errorf("UnlockOrder: %w", err)
	}
	if order.Status != domain.OrderSuspended {
		return fmt.Errorf("UnlockOrder: order is not suspended (current: %s)", order.Status)
	}

	// Resume at provider (best-effort)
	if order.ProviderOrderID != "" {
		provider := u.providerReg.Get(order.ProviderID.String())
		if provider != nil {
			if err := provider.Resume(ctx, order.ProviderOrderID); err != nil {
				u.logger.WarnContext(ctx, "UnlockOrder: resume at provider failed (continuing)", slog.String("err", err.Error()))
			}
		}
	}

	if err := u.orderRepo.UpdateStatus(ctx, req.OrderID, domain.OrderActive); err != nil {
		return fmt.Errorf("UnlockOrder: update status: %w", err)
	}
	_ = u.orderEvtRepo.Log(ctx, req.OrderID, domain.EventOrderUnlocked, map[string]any{
		"reason": req.Reason,
	})
	u.logger.InfoContext(ctx, "proxy unlocked by admin",
		slog.String("order_id", req.OrderID.String()),
		slog.String("reason", req.Reason),
	)
	return nil
}
