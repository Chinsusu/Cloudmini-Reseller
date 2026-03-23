// Package usecase — expiry lifecycle for proxy orders.
// expiry_usecase.go implements a background job that:
//  1. Moves 'active' orders past their expiry to 'expired' and suspends them at the provider.
//  2. After a configurable grace period, permanently deletes 'expired' orders from the provider.
package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/pvp/proxy-service/internal/domain"
	"github.com/pvp/proxy-service/internal/providers"
)

const (
	// DefaultGracePeriod is how long an expired proxy is kept suspended before permanent deletion.
	DefaultGracePeriod = 72 * time.Hour
)

// ExpiryUsecase drives the proxy expiry lifecycle.
type ExpiryUsecase struct {
	orderRepo    domain.IOrderRepository
	providerReg  *providers.Registry
	orderEvtRepo domain.IOrderEventRepository
	gracePeriod  time.Duration
	logger       *slog.Logger
}

// NewExpiryUsecase creates an ExpiryUsecase with the given grace period.
func NewExpiryUsecase(
	orderRepo domain.IOrderRepository,
	providerReg *providers.Registry,
	orderEvtRepo domain.IOrderEventRepository,
	gracePeriod time.Duration,
	logger *slog.Logger,
) *ExpiryUsecase {
	if gracePeriod == 0 {
		gracePeriod = DefaultGracePeriod
	}
	return &ExpiryUsecase{
		orderRepo:    orderRepo,
		providerReg:  providerReg,
		orderEvtRepo: orderEvtRepo,
		gracePeriod:  gracePeriod,
		logger:       logger,
	}
}

// ProcessExpired finds all 'active' orders past their expiry date, suspends them at the provider,
// and transitions them to 'expired'. Called every scheduling tick.
func (u *ExpiryUsecase) ProcessExpired(ctx context.Context) error {
	orders, err := u.orderRepo.ListExpiredActive(ctx)
	if err != nil {
		return fmt.Errorf("ExpiryUsecase.ProcessExpired: list: %w", err)
	}

	for _, o := range orders {
		log := u.logger.With("order_id", o.ID, "order_number", o.OrderNumber)

		// Suspend at provider (best-effort; log error but continue)
		if o.ProviderOrderID != "" {
			provider := u.providerReg.Get(o.ProviderID.String())
			if provider != nil {
				if err := provider.Suspend(ctx, o.ProviderOrderID); err != nil {
					log.Warn("ExpiryUsecase.ProcessExpired: suspend failed (continuing)", "err", err)
				}
			}
		}

		// Move to 'expired'
		if err := u.orderRepo.UpdateStatus(ctx, o.ID, domain.OrderExpired); err != nil {
			log.Error("ExpiryUsecase.ProcessExpired: update status", "err", err)
			continue
		}

		// Log event
		_ = u.orderEvtRepo.Log(ctx, o.ID, domain.EventOrderExpired, map[string]any{
			"grace_hours": u.gracePeriod.Hours(),
		})
		log.Info("proxy expired — suspended, grace period started")
	}
	return nil
}

// ProcessGraceExpired finds 'expired' orders whose grace period has elapsed, permanently deletes
// them from the provider, and transitions them to 'cancelled' (no refund). Called every tick.
func (u *ExpiryUsecase) ProcessGraceExpired(ctx context.Context) error {
	orders, err := u.orderRepo.ListExpiredGrace(ctx, u.gracePeriod)
	if err != nil {
		return fmt.Errorf("ExpiryUsecase.ProcessGraceExpired: list: %w", err)
	}

	for _, o := range orders {
		log := u.logger.With("order_id", o.ID, "order_number", o.OrderNumber)

		// Permanently delete from provider
		if o.ProviderOrderID != "" {
			provider := u.providerReg.Get(o.ProviderID.String())
			if provider != nil {
				if err := provider.Cancel(ctx, o.ProviderOrderID); err != nil {
					log.Warn("ExpiryUsecase.ProcessGraceExpired: cancel at provider failed (continuing)", "err", err)
				}
			}
		}

		// Move to 'cancelled' (permanent, no refund — proxy was already used)
		if err := u.orderRepo.UpdateStatus(ctx, o.ID, domain.OrderCancelled); err != nil {
			log.Error("ExpiryUsecase.ProcessGraceExpired: update status", "err", err)
			continue
		}

		// Log event
		_ = u.orderEvtRepo.Log(ctx, o.ID, domain.EventOrderDeleted, map[string]any{
			"reason": "grace_period_elapsed",
		})
		log.Info("proxy permanently deleted after grace period")
	}
	return nil
}
