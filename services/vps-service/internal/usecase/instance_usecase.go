package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/pvp/vps-service/internal/domain"
	"github.com/shopspring/decimal"
)

// InstanceUsecase handles instance lifecycle operations: start, stop, reboot, etc.
type InstanceUsecase struct {
	instanceRepo domain.IInstanceRepository
	snapshotRepo domain.ISnapshotRepository
	proxmox      ProxmoxAdapter
	eventPub     domain.IEventPublisher
	logger       *slog.Logger
}

// NewInstanceUsecase constructs InstanceUsecase.
func NewInstanceUsecase(
	instanceRepo domain.IInstanceRepository,
	snapshotRepo domain.ISnapshotRepository,
	proxmox ProxmoxAdapter,
	eventPub domain.IEventPublisher,
	logger *slog.Logger,
) *InstanceUsecase {
	return &InstanceUsecase{
		instanceRepo: instanceRepo,
		snapshotRepo: snapshotRepo,
		proxmox:      proxmox,
		eventPub:     eventPub,
		logger:       logger,
	}
}

func (u *InstanceUsecase) GetInstance(ctx context.Context, id, userID uuid.UUID) (*domain.Instance, error) {
	inst, err := u.instanceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("GetInstance: %w", err)
	}
	if inst.UserID != userID {
		return nil, domain.ErrInstanceNotOwned
	}
	return inst, nil
}

func (u *InstanceUsecase) ListInstances(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Instance, int, error) {
	return u.instanceRepo.ListByUser(ctx, userID, offset, limit)
}

func (u *InstanceUsecase) StartInstance(ctx context.Context, id, userID uuid.UUID) error {
	inst, err := u.getOwned(ctx, id, userID)
	if err != nil {
		return err
	}
	if inst.Status == domain.StatusTerminated {
		return domain.ErrInstanceTerminated
	}
	taskID, err := u.proxmox.StartVM(ctx, inst.NodeName, inst.VMID)
	if err != nil {
		return fmt.Errorf("StartInstance: %w", err)
	}
	if err := u.proxmox.WaitForTask(ctx, inst.NodeName, taskID, 30*time.Second); err != nil {
		return fmt.Errorf("StartInstance: wait: %w", err)
	}
	_ = u.instanceRepo.UpdateStatus(ctx, id, domain.StatusRunning)
	_ = u.eventPub.PublishStateChanged(ctx, id, inst.Status, domain.StatusRunning)
	return nil
}

func (u *InstanceUsecase) StopInstance(ctx context.Context, id, userID uuid.UUID) error {
	inst, err := u.getOwned(ctx, id, userID)
	if err != nil {
		return err
	}
	taskID, err := u.proxmox.ShutdownVM(ctx, inst.NodeName, inst.VMID)
	if err != nil {
		return fmt.Errorf("StopInstance: %w", err)
	}
	if err := u.proxmox.WaitForTask(ctx, inst.NodeName, taskID, 60*time.Second); err != nil {
		return fmt.Errorf("StopInstance: wait: %w", err)
	}
	_ = u.instanceRepo.UpdateStatus(ctx, id, "stopped")
	_ = u.eventPub.PublishStateChanged(ctx, id, inst.Status, "stopped")
	return nil
}

func (u *InstanceUsecase) RebootInstance(ctx context.Context, id, userID uuid.UUID) error {
	inst, err := u.getOwned(ctx, id, userID)
	if err != nil {
		return err
	}
	if inst.Status != domain.StatusRunning {
		return domain.ErrInstanceNotRunning
	}
	taskID, err := u.proxmox.RebootVM(ctx, inst.NodeName, inst.VMID)
	if err != nil {
		return fmt.Errorf("RebootInstance: %w", err)
	}
	_ = u.proxmox.WaitForTask(ctx, inst.NodeName, taskID, 60*time.Second)
	return nil
}

// SuspendInstance is called by billing cron when wallet is empty.
func (u *InstanceUsecase) SuspendInstance(ctx context.Context, id uuid.UUID, reason string) error {
	inst, err := u.instanceRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("SuspendInstance: %w", err)
	}
	taskID, err := u.proxmox.SuspendVM(ctx, inst.NodeName, inst.VMID)
	if err != nil {
		return fmt.Errorf("SuspendInstance: %w", err)
	}
	_ = u.proxmox.WaitForTask(ctx, inst.NodeName, taskID, 60*time.Second)
	_ = u.instanceRepo.UpdateStatus(ctx, id, domain.StatusSuspended)
	_ = u.eventPub.PublishStateChanged(ctx, id, inst.Status, domain.StatusSuspended)
	_ = u.eventPub.PublishSuspended(ctx, id, reason)
	return nil
}

func (u *InstanceUsecase) ResumeInstance(ctx context.Context, id, userID uuid.UUID) error {
	inst, err := u.getOwned(ctx, id, userID)
	if err != nil {
		return err
	}
	if inst.Status != domain.StatusSuspended {
		return domain.ErrInstanceNotSuspended
	}
	taskID, err := u.proxmox.ResumeVM(ctx, inst.NodeName, inst.VMID)
	if err != nil {
		return fmt.Errorf("ResumeInstance: %w", err)
	}
	_ = u.proxmox.WaitForTask(ctx, inst.NodeName, taskID, 60*time.Second)
	_ = u.instanceRepo.UpdateStatus(ctx, id, domain.StatusRunning)
	_ = u.eventPub.PublishStateChanged(ctx, id, inst.Status, domain.StatusRunning)
	return nil
}

func (u *InstanceUsecase) TerminateInstance(ctx context.Context, id, userID uuid.UUID) error {
	inst, err := u.getOwned(ctx, id, userID)
	if err != nil {
		return err
	}
	if inst.Status == domain.StatusTerminated {
		return domain.ErrInstanceTerminated
	}

	// Stop first if running
	if inst.Status == domain.StatusRunning {
		_, _ = u.proxmox.StopVM(ctx, inst.NodeName, inst.VMID)
		time.Sleep(5 * time.Second)
	}

	taskID, err := u.proxmox.DeleteVM(ctx, inst.NodeName, inst.VMID)
	if err != nil {
		return fmt.Errorf("TerminateInstance: delete vm: %w", err)
	}
	_ = u.proxmox.WaitForTask(ctx, inst.NodeName, taskID, 60*time.Second)
	_ = u.instanceRepo.UpdateStatus(ctx, id, domain.StatusTerminated)
	_ = u.instanceRepo.SoftDelete(ctx, id)
	_ = u.eventPub.PublishTerminated(ctx, id, userID)
	return nil
}

func (u *InstanceUsecase) GetConsoleURL(ctx context.Context, id, userID uuid.UUID) (string, error) {
	inst, err := u.getOwned(ctx, id, userID)
	if err != nil {
		return "", err
	}
	url, err := u.proxmox.GetConsoleURL(ctx, inst.NodeName, inst.VMID)
	if err != nil {
		return "", fmt.Errorf("GetConsoleURL: %w", err)
	}
	return url, nil
}

// ─── Snapshots ────────────────────────────────────────────────────────────────

func (u *InstanceUsecase) CreateSnapshot(ctx context.Context, instID, userID uuid.UUID, name, desc string) (*domain.Snapshot, error) {
	inst, err := u.getOwned(ctx, instID, userID)
	if err != nil {
		return nil, err
	}
	proxName := fmt.Sprintf("snap-%d", time.Now().Unix())
	taskID, err := u.proxmox.CreateSnapshot(ctx, inst.NodeName, inst.VMID, proxName, desc)
	if err != nil {
		return nil, fmt.Errorf("CreateSnapshot: %w", err)
	}
	_ = u.proxmox.WaitForTask(ctx, inst.NodeName, taskID, 120*time.Second)

	snap := &domain.Snapshot{
		ID: uuid.New(), InstanceID: instID,
		Name: name, ProxmoxName: proxName, Description: desc,
		CreatedAt: time.Now(),
	}
	if err := u.snapshotRepo.Create(ctx, snap); err != nil {
		return nil, fmt.Errorf("CreateSnapshot: save: %w", err)
	}
	return snap, nil
}

func (u *InstanceUsecase) ListSnapshots(ctx context.Context, instID, userID uuid.UUID) ([]*domain.Snapshot, error) {
	if _, err := u.getOwned(ctx, instID, userID); err != nil {
		return nil, err
	}
	return u.snapshotRepo.ListByInstance(ctx, instID)
}

func (u *InstanceUsecase) DeleteSnapshot(ctx context.Context, instID, snapID, userID uuid.UUID) error {
	inst, err := u.getOwned(ctx, instID, userID)
	if err != nil {
		return err
	}
	snap, err := u.snapshotRepo.GetByID(ctx, snapID)
	if err != nil {
		return domain.ErrSnapshotNotFound
	}
	taskID, err := u.proxmox.DeleteSnapshot(ctx, inst.NodeName, inst.VMID, snap.ProxmoxName)
	if err != nil {
		return fmt.Errorf("DeleteSnapshot: %w", err)
	}
	_ = u.proxmox.WaitForTask(ctx, inst.NodeName, taskID, 60*time.Second)
	return u.snapshotRepo.Delete(ctx, snapID)
}

func (u *InstanceUsecase) getOwned(ctx context.Context, id, userID uuid.UUID) (*domain.Instance, error) {
	inst, err := u.instanceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getOwned: %w", err)
	}
	if inst.UserID != userID {
		return nil, domain.ErrInstanceNotOwned
	}
	return inst, nil
}

// ─── Billing Cron ─────────────────────────────────────────────────────────────

// BillingCron handles hourly usage metering and auto-suspend on wallet.empty.
type BillingCron struct {
	instanceRepo domain.IInstanceRepository
	planRepo     domain.IPlanRepository
	instanceUC   *InstanceUsecase
	eventPub     domain.IEventPublisher
	logger       *slog.Logger
}

// NewBillingCron constructs BillingCron.
func NewBillingCron(
	instanceRepo domain.IInstanceRepository,
	planRepo domain.IPlanRepository,
	instanceUC *InstanceUsecase,
	eventPub domain.IEventPublisher,
	logger *slog.Logger,
) *BillingCron {
	return &BillingCron{
		instanceRepo: instanceRepo,
		planRepo:     planRepo,
		instanceUC:   instanceUC,
		eventPub:     eventPub,
		logger:       logger,
	}
}

// RunHourlyMeter emits vps.usage.report for every running instance.
// Call this every hour from a cron goroutine.
func (c *BillingCron) RunHourlyMeter(ctx context.Context) error {
	instances, err := c.instanceRepo.ListRunning(ctx)
	if err != nil {
		return fmt.Errorf("BillingCron.RunHourlyMeter: list: %w", err)
	}

	now := time.Now()
	for _, inst := range instances {
		plan, err := c.planRepo.GetByID(ctx, inst.PlanID)
		if err != nil {
			c.logger.Warn("billing cron: get plan", slog.String("plan_id", inst.PlanID.String()), slog.String("error", err.Error()))
			continue
		}

		var hours float64 = 1.0
		if inst.LastBilledAt != nil {
			hours = now.Sub(*inst.LastBilledAt).Hours()
			if hours < 0.01 {
				continue // too soon
			}
		}

		amount := plan.HourlyRate.Mul(decimal.NewFromFloat(hours))

		if err := c.eventPub.PublishUsageReport(ctx, inst.ID, hours, amount); err != nil {
			c.logger.Warn("billing cron: publish usage", slog.String("instance_id", inst.ID.String()), slog.String("error", err.Error()))
			continue
		}

		if err := c.instanceRepo.UpdateLastBilled(ctx, inst.ID, now); err != nil {
			c.logger.Warn("billing cron: update last_billed", slog.String("instance_id", inst.ID.String()), slog.String("error", err.Error()))
		}

		c.logger.Info("usage metered",
			slog.String("instance_id", inst.ID.String()),
			slog.Float64("hours", hours),
			slog.String("amount", amount.String()),
		)
	}
	return nil
}

// HandleWalletEmpty suspends all running VPS instances for a user.
// Triggered by billing.wallet.empty NATS event.
func (c *BillingCron) HandleWalletEmpty(ctx context.Context, userID uuid.UUID) error {
	instances, _, err := c.instanceRepo.ListByUser(ctx, userID, 0, 100)
	if err != nil {
		return fmt.Errorf("HandleWalletEmpty: list: %w", err)
	}

	for _, inst := range instances {
		if inst.Status != domain.StatusRunning {
			continue
		}
		if err := c.instanceUC.SuspendInstance(ctx, inst.ID, "wallet_empty"); err != nil {
			c.logger.Warn("HandleWalletEmpty: suspend failed",
				slog.String("instance_id", inst.ID.String()),
				slog.String("error", err.Error()),
			)
		}
	}
	return nil
}

// StartHourlyCron runs the billing meter every hour in a background goroutine.
func (c *BillingCron) StartHourlyCron(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.RunHourlyMeter(ctx); err != nil {
					c.logger.Error("hourly billing meter", slog.String("error", err.Error()))
				}
			}
		}
	}()
}
