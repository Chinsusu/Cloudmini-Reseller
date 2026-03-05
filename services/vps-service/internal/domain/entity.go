// Package domain contains core entities, repository interfaces, and errors
// for vps-service.
package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Instance status constants — follows the state machine in VPS-SERVICE spec.
const (
	StatusPending        = "pending"
	StatusProvisioning   = "provisioning"
	StatusBooting        = "booting"
	StatusRunning        = "running"
	StatusSuspended      = "suspended"
	StatusTerminated     = "terminated"
	StatusFailed         = "failed"
)

// Plan represents a VPS plan (e.g. "Starter 1", "GPU Pro").
type Plan struct {
	ID          uuid.UUID       `db:"id"`
	Name        string          `db:"name"`
	CPU         int             `db:"cpu"`
	RAMMB       int             `db:"ram_mb"`
	DiskGB      int             `db:"disk_gb"`
	BandwidthGB int             `db:"bandwidth_gb"`
	OSType      string          `db:"os_type"`
	Template    string          `db:"template"`
	HourlyRate  decimal.Decimal `db:"hourly_rate"`
	MonthlyRate decimal.Decimal `db:"monthly_rate"`
	IsActive    bool            `db:"is_active"`
	CreatedAt   time.Time       `db:"created_at"`
}

// Node represents a Proxmox node managed by the platform.
type Node struct {
	ID           uuid.UUID       `db:"id"`
	Name         string          `db:"name"`         // pve1, pve2, ...
	DisplayName  string          `db:"display_name"`
	Location     string          `db:"location"`
	ProxmoxHost  string          `db:"proxmox_host"`
	ProxmoxPort  int             `db:"proxmox_port"`
	TotalCPU     int             `db:"total_cpu"`
	TotalRAMMB   int             `db:"total_ram_mb"`
	TotalDiskGB  int             `db:"total_disk_gb"`
	ReservedCPU  int             `db:"reserved_cpu"`
	ReservedRAMMB int            `db:"reserved_ram_mb"`
	ReservedDiskGB int           `db:"reserved_disk_gb"`
	Status       string          `db:"status"` // online|maintenance|offline
	CreatedAt    time.Time       `db:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at"`
}

// AvailableRAMMB returns available RAM with 20% headroom for the given plan.
func (n *Node) AvailableRAMMB() int { return n.TotalRAMMB - n.ReservedRAMMB }

// Instance represents a user's VPS instance.
type Instance struct {
	ID              uuid.UUID       `db:"id"`
	UserID          uuid.UUID       `db:"user_id"`
	ResellerID      *uuid.UUID      `db:"reseller_id"`
	PlanID          uuid.UUID       `db:"plan_id"`
	NodeID          uuid.UUID       `db:"node_id"`
	NodeName        string          `db:"node_name"`
	VMID            int             `db:"vmid"`
	Hostname        string          `db:"hostname"`
	Status          string          `db:"status"`
	IPAddress       string          `db:"ip_address"`
	SSHPort         int             `db:"ssh_port"`
	RootPassword    string          `db:"root_password"` // encrypted
	BillingStartedAt *time.Time     `db:"billing_started_at"`
	LastBilledAt    *time.Time      `db:"last_billed_at"`
	SuspendedAt     *time.Time      `db:"suspended_at"`
	TerminatedAt    *time.Time      `db:"terminated_at"`
	IdempotencyKey  string          `db:"idempotency_key"`
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
}

// Snapshot represents a VM snapshot.
type Snapshot struct {
	ID         uuid.UUID  `db:"id"`
	InstanceID uuid.UUID  `db:"instance_id"`
	Name       string     `db:"name"`
	ProxmoxName string    `db:"proxmox_name"`
	Description string    `db:"description"`
	SizeGB     decimal.Decimal `db:"size_gb"`
	CreatedAt  time.Time  `db:"created_at"`
}

// ─── Repository Interfaces ────────────────────────────────────────────────────

// IPlanRepository manages VPS plans.
type IPlanRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Plan, error)
	List(ctx context.Context) ([]*Plan, error)
}

// INodeRepository manages Proxmox nodes.
type INodeRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Node, error)
	GetByName(ctx context.Context, name string) (*Node, error)
	ListOnline(ctx context.Context) ([]*Node, error)
	ReserveResources(ctx context.Context, nodeID uuid.UUID, cpu, ramMB, diskGB int) error
	ReleaseResources(ctx context.Context, nodeID uuid.UUID, cpu, ramMB, diskGB int) error
}

// IInstanceRepository manages VPS instances.
type IInstanceRepository interface {
	Create(ctx context.Context, inst *Instance) error
	GetByID(ctx context.Context, id uuid.UUID) (*Instance, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*Instance, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdateAfterProvisioning(ctx context.Context, id uuid.UUID, vmid int, ipAddress, rootPassword string, billingStartedAt time.Time) error
	UpdateLastBilled(ctx context.Context, id uuid.UUID, t time.Time) error
	ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*Instance, int, error)
	ListRunning(ctx context.Context) ([]*Instance, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

// ISnapshotRepository manages instance snapshots.
type ISnapshotRepository interface {
	Create(ctx context.Context, s *Snapshot) error
	GetByID(ctx context.Context, id uuid.UUID) (*Snapshot, error)
	ListByInstance(ctx context.Context, instanceID uuid.UUID) ([]*Snapshot, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// IEventPublisher publishes VPS events to NATS.
type IEventPublisher interface {
	PublishStateChanged(ctx context.Context, instanceID uuid.UUID, from, to string) error
	PublishProvisionRequested(ctx context.Context, inst *Instance) error
	PublishVMReady(ctx context.Context, inst *Instance) error
	PublishProvisionFailed(ctx context.Context, instanceID uuid.UUID, reason, step string) error
	PublishSuspended(ctx context.Context, instanceID uuid.UUID, reason string) error
	PublishTerminated(ctx context.Context, instanceID, userID uuid.UUID) error
	PublishUsageReport(ctx context.Context, instanceID uuid.UUID, hours float64, amount decimal.Decimal) error
}
