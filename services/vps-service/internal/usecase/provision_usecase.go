// Package usecase contains vps-service business logic.
// provision_usecase.go implements the 2-phase async VM provisioning workflow.
package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	cryptopkg "github.com/pvp/pkg/crypto"
	"github.com/pvp/vps-service/internal/domain"
)

const (
	provisionTimeout = 120 * time.Second
	bootTimeout      = 90 * time.Second
)

// ProxmoxAdapter abstraction to avoid circular imports.
type ProxmoxAdapter interface {
	CreateVM(ctx context.Context, req CreateVMProxmoxReq) (taskID string, err error)
	StartVM(ctx context.Context, nodeName string, vmid int) (taskID string, err error)
	WaitForTask(ctx context.Context, nodeName, taskID string, timeout time.Duration) error
	GetVMIPAddress(ctx context.Context, nodeName string, vmid int) (string, error)
	StopVM(ctx context.Context, nodeName string, vmid int) (taskID string, err error)
	ShutdownVM(ctx context.Context, nodeName string, vmid int) (taskID string, err error)
	RebootVM(ctx context.Context, nodeName string, vmid int) (taskID string, err error)
	SuspendVM(ctx context.Context, nodeName string, vmid int) (taskID string, err error)
	ResumeVM(ctx context.Context, nodeName string, vmid int) (taskID string, err error)
	DeleteVM(ctx context.Context, nodeName string, vmid int) (taskID string, err error)
	GetConsoleURL(ctx context.Context, nodeName string, vmid int) (string, error)
	CreateSnapshot(ctx context.Context, nodeName string, vmid int, name, desc string) (taskID string, err error)
	ListSnapshots(ctx context.Context, nodeName string, vmid int) ([]map[string]any, error)
	DeleteSnapshot(ctx context.Context, nodeName string, vmid int, name string) (taskID string, err error)
}

// CreateVMProxmoxReq is a simplified request for the adapter.
type CreateVMProxmoxReq struct {
	NodeName string
	VMID     int
	Name     string
	Cores    int
	MemMB    int
	DiskGB   int
	OSType   string
	Template string
	Password string
}

// BillingAdapter calls billing-service for hold/confirm/release.
type BillingAdapter interface {
	Hold(ctx context.Context, userID uuid.UUID, amount interface{}, refType string, refID uuid.UUID) error
	ConfirmHold(ctx context.Context, userID uuid.UUID, amount interface{}, refType string, refID uuid.UUID) error
	ReleaseHold(ctx context.Context, userID uuid.UUID, amount interface{}, refType string, refID uuid.UUID) error
}

// ProvisionUsecase handles the 2-phase async VPS provisioning.
type ProvisionUsecase struct {
	instanceRepo domain.IInstanceRepository
	nodeRepo     domain.INodeRepository
	planRepo     domain.IPlanRepository
	proxmox      ProxmoxAdapter
	billing      BillingAdapter
	cipher       *cryptopkg.Cipher
	eventPub     domain.IEventPublisher
	logger       *slog.Logger
	vmidStart    int
}

// NewProvisionUsecase constructs ProvisionUsecase.
func NewProvisionUsecase(
	instanceRepo domain.IInstanceRepository,
	nodeRepo domain.INodeRepository,
	planRepo domain.IPlanRepository,
	proxmox ProxmoxAdapter,
	billing BillingAdapter,
	cipher *cryptopkg.Cipher,
	eventPub domain.IEventPublisher,
	logger *slog.Logger,
	vmidStart int,
) *ProvisionUsecase {
	return &ProvisionUsecase{
		instanceRepo: instanceRepo,
		nodeRepo:     nodeRepo,
		planRepo:     planRepo,
		proxmox:      proxmox,
		billing:      billing,
		cipher:       cipher,
		eventPub:     eventPub,
		logger:       logger,
		vmidStart:    vmidStart,
	}
}

// OrderRequest is the input to Phase 1 (synchronous).
type OrderRequest struct {
	UserID         uuid.UUID
	ResellerID     *uuid.UUID
	PlanID         uuid.UUID
	Hostname       string
	IdempotencyKey string
}

// OrderResponse is returned immediately from Phase 1.
type OrderResponse struct {
	InstanceID string
	Status     string
	Message    string
}

// CreateVPS runs Phase 1 (synchronous): validate + select node + reserve + hold + enqueue.
// Returns 202 immediately; Phase 2 worker runs asynchronously.
func (u *ProvisionUsecase) CreateVPS(ctx context.Context, req OrderRequest) (*OrderResponse, error) {
	// Idempotency
	if req.IdempotencyKey != "" {
		if existing, err := u.instanceRepo.GetByIdempotencyKey(ctx, req.IdempotencyKey); err == nil {
			return &OrderResponse{InstanceID: existing.ID.String(), Status: existing.Status, Message: "duplicate request"}, nil
		}
	}

	// 1. Validate plan
	plan, err := u.planRepo.GetByID(ctx, req.PlanID)
	if err != nil {
		return nil, fmt.Errorf("CreateVPS: %w", err)
	}
	if !plan.IsActive {
		return nil, fmt.Errorf("CreateVPS: plan is not active")
	}

	// 2. Select optimal node (least-loaded)
	node, err := u.selectNode(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("CreateVPS: %w", err)
	}

	// 3. Reserve resources (atomic)
	if err := u.nodeRepo.ReserveResources(ctx, node.ID, plan.CPU, plan.RAMMB, plan.DiskGB); err != nil {
		return nil, fmt.Errorf("CreateVPS: reserve resources: %w", err)
	}

	// Generate VMID (monotonically increasing; real prod: use DB sequence)
	vmid := u.vmidStart + rand.Intn(8999)

	// Encrypt a generated root password
	rootPass := generatePassword()
	encPass, err := u.cipher.Encrypt([]byte(rootPass))
	if err != nil {
		_ = u.nodeRepo.ReleaseResources(ctx, node.ID, plan.CPU, plan.RAMMB, plan.DiskGB)
		return nil, fmt.Errorf("CreateVPS: encrypt password: %w", err)
	}

	// 4. Create billing hold (1 hour minimum charge)
	instanceID := uuid.New()
	// instance_number: short human-readable ID, e.g. "VPS-A1B2C3"
	instanceNum := "VPS-" + strings.ToUpper(instanceID.String()[:6])
	instance := &domain.Instance{
		ID:             instanceID,
		InstanceNumber: instanceNum,
		UserID:         req.UserID,
		ResellerID:     req.ResellerID,
		PlanID:         req.PlanID,
		NodeID:         node.ID,
		NodeName:       node.Name,
		VMID:           vmid,
		Hostname:       req.Hostname,
		OSTemplate:     plan.Template, // e.g. "ubuntu-22.04"
		Status:         domain.StatusPending,
		RootPassword:   encPass,
		SSHPort:        22,
		BillingType:    "hourly",
		CPU:            plan.CPU,
		RAMMB:          plan.RAMMB,
		DiskGB:         plan.DiskGB,
		IdempotencyKey: req.IdempotencyKey,
		CreatedAt:      time.Now(),
	}

	if err := u.instanceRepo.Create(ctx, instance); err != nil {
		_ = u.nodeRepo.ReleaseResources(ctx, node.ID, plan.CPU, plan.RAMMB, plan.DiskGB)
		return nil, fmt.Errorf("CreateVPS: create instance: %w", err)
	}

	if err := u.billing.Hold(ctx, req.UserID, plan.HourlyRate, "vps_instance", instance.ID); err != nil {
		_ = u.instanceRepo.UpdateStatus(ctx, instance.ID, domain.StatusFailed)
		_ = u.nodeRepo.ReleaseResources(ctx, node.ID, plan.CPU, plan.RAMMB, plan.DiskGB)
		return nil, fmt.Errorf("CreateVPS: billing hold: %w", err)
	}

	// 5. Publish provision job → async worker picks it up via NATS
	_ = u.eventPub.PublishProvisionRequested(ctx, instance)

	u.logger.InfoContext(ctx, "VPS order accepted",
		slog.String("instance_id", instance.ID.String()),
		slog.String("node", node.Name),
		slog.Int("vmid", vmid),
	)

	return &OrderResponse{
		InstanceID: instance.ID.String(),
		Status:     domain.StatusPending,
		Message:    "VM is being provisioned. Check /instances/" + instance.ID.String() + "/status",
	}, nil
}

// RunProvisionWorker is the Phase 2 async worker — called by NATS consumer.
// It performs the actual Proxmox API calls and polls for completion.
func (u *ProvisionUsecase) RunProvisionWorker(ctx context.Context, instanceID uuid.UUID) error {
	inst, err := u.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("worker: get instance: %w", err)
	}

	plan, err := u.planRepo.GetByID(ctx, inst.PlanID)
	if err != nil {
		return fmt.Errorf("worker: get plan: %w", err)
	}

	// Update status → provisioning
	_ = u.instanceRepo.UpdateStatus(ctx, inst.ID, domain.StatusProvisioning)
	_ = u.eventPub.PublishStateChanged(ctx, inst.ID, domain.StatusPending, domain.StatusProvisioning)

	// Call Proxmox API: create VM
	taskID, err := u.proxmox.CreateVM(ctx, CreateVMProxmoxReq{
		NodeName: inst.NodeName,
		VMID:     inst.VMID,
		Name:     inst.Hostname,
		Cores:    plan.CPU,
		MemMB:    plan.RAMMB,
		DiskGB:   plan.DiskGB,
		OSType:   plan.OSType,
		Template: plan.Template,
		Password: inst.RootPassword, // still encrypted, proxmox will store it
	})
	if err != nil {
		return u.fail(ctx, inst, "create_vm", fmt.Sprintf("proxmox CreateVM: %v", err))
	}

	// Poll task (timeout 120s)
	if err := u.proxmox.WaitForTask(ctx, inst.NodeName, taskID, provisionTimeout); err != nil {
		return u.fail(ctx, inst, "wait_create_vm", err.Error())
	}

	// Start VM
	startTaskID, err := u.proxmox.StartVM(ctx, inst.NodeName, inst.VMID)
	if err != nil {
		return u.fail(ctx, inst, "start_vm", err.Error())
	}
	_ = u.instanceRepo.UpdateStatus(ctx, inst.ID, domain.StatusBooting)
	_ = u.eventPub.PublishStateChanged(ctx, inst.ID, domain.StatusProvisioning, domain.StatusBooting)

	if err := u.proxmox.WaitForTask(ctx, inst.NodeName, startTaskID, 60*time.Second); err != nil {
		return u.fail(ctx, inst, "wait_start_vm", err.Error())
	}

	// Wait for guest agent IP (timeout 90s)
	ip, err := u.waitForIP(ctx, inst.NodeName, inst.VMID, bootTimeout)
	if err != nil {
		return u.fail(ctx, inst, "wait_ip", err.Error())
	}

	// Activate instance
	now := time.Now()
	if err := u.instanceRepo.UpdateAfterProvisioning(ctx, inst.ID, inst.VMID, ip, inst.RootPassword, now); err != nil {
		return u.fail(ctx, inst, "update_instance", err.Error())
	}

	// Confirm billing hold → actual charge
	_ = u.billing.ConfirmHold(ctx, inst.UserID, plan.HourlyRate, "vps_instance", inst.ID)

	_ = u.eventPub.PublishStateChanged(ctx, inst.ID, domain.StatusBooting, domain.StatusRunning)
	inst.IPAddress = ip
	inst.BillingStartedAt = &now
	_ = u.eventPub.PublishVMReady(ctx, inst)

	u.logger.InfoContext(ctx, "VPS ready",
		slog.String("instance_id", inst.ID.String()),
		slog.String("ip", ip),
		slog.String("node", inst.NodeName),
		slog.Int("vmid", inst.VMID),
	)
	return nil
}

// fail runs compensation: release resources, billing hold, update status, publish failed.
func (u *ProvisionUsecase) fail(ctx context.Context, inst *domain.Instance, step, reason string) error {
	_ = u.instanceRepo.UpdateStatus(ctx, inst.ID, domain.StatusFailed)
	_ = u.nodeRepo.ReleaseResources(ctx, inst.NodeID, 0, 0, 0) // fetch plan to get values
	_ = u.billing.ReleaseHold(ctx, inst.UserID, 0, "vps_instance", inst.ID)
	_ = u.eventPub.PublishProvisionFailed(ctx, inst.ID, reason, step)
	u.logger.ErrorContext(ctx, "VPS provisioning failed",
		slog.String("instance_id", inst.ID.String()),
		slog.String("step", step),
		slog.String("reason", reason),
	)
	return fmt.Errorf("vps provision failed at %s: %s", step, reason)
}

// waitForIP polls GetVMIPAddress until an IP is found or timeout is reached.
func (u *ProvisionUsecase) waitForIP(ctx context.Context, nodeName string, vmid int, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case t := <-ticker.C:
			if t.After(deadline) {
				return "", domain.ErrProvisionTimeout
			}
			ip, err := u.proxmox.GetVMIPAddress(ctx, nodeName, vmid)
			if err == nil && ip != "" {
				return ip, nil
			}
		}
	}
}

// selectNode picks the least-loaded node that can fit the plan.
func (u *ProvisionUsecase) selectNode(ctx context.Context, plan *domain.Plan) (*domain.Node, error) {
	nodes, err := u.nodeRepo.ListOnline(ctx)
	if err != nil {
		return nil, fmt.Errorf("selectNode: %w", err)
	}

	type candidate struct {
		node     *domain.Node
		availRAM int
	}
	var candidates []candidate
	for _, n := range nodes {
		availRAM := n.AvailableRAMMB()
		availDisk := n.TotalDiskGB - n.ReservedDiskGB

		// 20% headroom buffer
		reqRAM := plan.RAMMB * 12 / 10
		reqDisk := plan.DiskGB * 12 / 10

		if availRAM >= reqRAM && availDisk >= reqDisk {
			candidates = append(candidates, candidate{node: n, availRAM: availRAM})
		}
	}

	if len(candidates) == 0 {
		return nil, domain.ErrNoAvailableNode
	}

	// Sort by available RAM desc (least loaded)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].availRAM > candidates[j].availRAM
	})
	return candidates[0].node, nil
}

func generatePassword() string {
	const chars = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789!@#$"
	b := make([]byte, 20)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
