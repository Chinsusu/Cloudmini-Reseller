package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pvp/vps-service/internal/domain"
)

// ─── PlanRepository ───────────────────────────────────────────────────────────

type PlanRepository struct{ db *sqlx.DB }

func NewPlanRepository(db *sqlx.DB) *PlanRepository { return &PlanRepository{db: db} }

const planCols = `id, name, slug, cpu_cores, ram_mb, disk_gb,
	COALESCE(bandwidth_gb, 0) AS bandwidth_gb,
	hourly_rate, COALESCE(monthly_rate, 0) AS monthly_rate,
	is_active, created_at`

func (r *PlanRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Plan, error) {
	var p domain.Plan
	q := `SELECT ` + planCols + ` FROM vps.plans WHERE id=$1`
	if err := r.db.GetContext(ctx, &p, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPlanNotFound
		}
		return nil, fmt.Errorf("PlanRepository.GetByID: %w", err)
	}
	return &p, nil
}

func (r *PlanRepository) List(ctx context.Context) ([]*domain.Plan, error) {
	var plans []*domain.Plan
	q := `SELECT ` + planCols + ` FROM vps.plans WHERE is_active=true ORDER BY monthly_rate ASC`
	if err := r.db.SelectContext(ctx, &plans, q); err != nil {
		return nil, fmt.Errorf("PlanRepository.List: %w", err)
	}
	return plans, nil
}

func (r *PlanRepository) Create(ctx context.Context, p *domain.Plan) error {
	q := `INSERT INTO vps.plans
		(id,name,slug,cpu_cores,ram_mb,disk_gb,bandwidth_gb,hourly_rate,monthly_rate,is_active,created_at)
		VALUES (:id,:name,:slug,:cpu_cores,:ram_mb,:disk_gb,:bandwidth_gb,:hourly_rate,:monthly_rate,:is_active,NOW())`
	if _, err := r.db.NamedExecContext(ctx, q, p); err != nil {
		return fmt.Errorf("PlanRepository.Create: %w", err)
	}
	return nil
}

func (r *PlanRepository) ToggleActive(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE vps.plans SET is_active = NOT is_active WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("PlanRepository.ToggleActive: %w", err)
	}
	return nil
}

// ─── NodeRepository ───────────────────────────────────────────────────────────

type NodeRepository struct{ db *sqlx.DB }

func NewNodeRepository(db *sqlx.DB) *NodeRepository { return &NodeRepository{db: db} }

func (r *NodeRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Node, error) {
	var n domain.Node
	if err := r.db.GetContext(ctx, &n, `SELECT * FROM vps.nodes WHERE id=$1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNodeNotFound
		}
		return nil, fmt.Errorf("NodeRepository.GetByID: %w", err)
	}
	return &n, nil
}

func (r *NodeRepository) GetByName(ctx context.Context, name string) (*domain.Node, error) {
	var n domain.Node
	if err := r.db.GetContext(ctx, &n, `SELECT * FROM vps.nodes WHERE name=$1`, name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNodeNotFound
		}
		return nil, fmt.Errorf("NodeRepository.GetByName: %w", err)
	}
	return &n, nil
}

func (r *NodeRepository) ListOnline(ctx context.Context) ([]*domain.Node, error) {
	var nodes []*domain.Node
	if err := r.db.SelectContext(ctx, &nodes, `SELECT * FROM vps.nodes WHERE status='online' ORDER BY reserved_ram_mb ASC`); err != nil {
		return nil, fmt.Errorf("NodeRepository.ListOnline: %w", err)
	}
	return nodes, nil
}

func (r *NodeRepository) ReserveResources(ctx context.Context, nodeID uuid.UUID, cpu, ramMB, diskGB int) error {
	q := `UPDATE vps.nodes SET
		reserved_cpu = reserved_cpu + $1,
		reserved_ram_mb = reserved_ram_mb + $2,
		reserved_disk_gb = reserved_disk_gb + $3,
		updated_at = NOW()
	WHERE id = $4`
	if _, err := r.db.ExecContext(ctx, q, cpu, ramMB, diskGB, nodeID); err != nil {
		return fmt.Errorf("NodeRepository.ReserveResources: %w", err)
	}
	return nil
}

func (r *NodeRepository) ReleaseResources(ctx context.Context, nodeID uuid.UUID, cpu, ramMB, diskGB int) error {
	q := `UPDATE vps.nodes SET
		reserved_cpu = GREATEST(0, reserved_cpu - $1),
		reserved_ram_mb = GREATEST(0, reserved_ram_mb - $2),
		reserved_disk_gb = GREATEST(0, reserved_disk_gb - $3),
		updated_at = NOW()
	WHERE id = $4`
	if _, err := r.db.ExecContext(ctx, q, cpu, ramMB, diskGB, nodeID); err != nil {
		return fmt.Errorf("NodeRepository.ReleaseResources: %w", err)
	}
	return nil
}

// ─── InstanceRepository ───────────────────────────────────────────────────────

type InstanceRepository struct{ db *sqlx.DB }

func NewInstanceRepository(db *sqlx.DB) *InstanceRepository { return &InstanceRepository{db: db} }

func (r *InstanceRepository) Create(ctx context.Context, inst *domain.Instance) error {
	q := `INSERT INTO vps.instances
		(id,user_id,reseller_id,plan_id,node_id,node_name,vmid,hostname,status,
		 root_password,idempotency_key,ssh_port,created_at,updated_at)
		VALUES (:id,:user_id,:reseller_id,:plan_id,:node_id,:node_name,:vmid,:hostname,
		 :status,:root_password,:idempotency_key,:ssh_port,:created_at,:updated_at)`
	if _, err := r.db.NamedExecContext(ctx, q, inst); err != nil {
		return fmt.Errorf("InstanceRepository.Create: %w", err)
	}
	return nil
}

func (r *InstanceRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Instance, error) {
	var inst domain.Instance
	if err := r.db.GetContext(ctx, &inst, `SELECT * FROM vps.instances WHERE id=$1 AND terminated_at IS NULL`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrInstanceNotFound
		}
		return nil, fmt.Errorf("InstanceRepository.GetByID: %w", err)
	}
	return &inst, nil
}

func (r *InstanceRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Instance, error) {
	var inst domain.Instance
	if err := r.db.GetContext(ctx, &inst, `SELECT * FROM vps.instances WHERE idempotency_key=$1`, key); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrInstanceNotFound
		}
		return nil, fmt.Errorf("InstanceRepository.GetByIdempotencyKey: %w", err)
	}
	return &inst, nil
}

func (r *InstanceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if _, err := r.db.ExecContext(ctx,
		`UPDATE vps.instances SET status=$1, updated_at=NOW() WHERE id=$2`, status, id,
	); err != nil {
		return fmt.Errorf("InstanceRepository.UpdateStatus: %w", err)
	}
	return nil
}

func (r *InstanceRepository) UpdateAfterProvisioning(ctx context.Context, id uuid.UUID, vmid int, ip, rootPass string, billedAt time.Time) error {
	q := `UPDATE vps.instances SET status='running', vmid=$1, ip_address=$2, root_password=$3,
		billing_started_at=$4, last_billed_at=$4, updated_at=NOW() WHERE id=$5`
	if _, err := r.db.ExecContext(ctx, q, vmid, ip, rootPass, billedAt, id); err != nil {
		return fmt.Errorf("InstanceRepository.UpdateAfterProvisioning: %w", err)
	}
	return nil
}

func (r *InstanceRepository) UpdateLastBilled(ctx context.Context, id uuid.UUID, t time.Time) error {
	if _, err := r.db.ExecContext(ctx,
		`UPDATE vps.instances SET last_billed_at=$1, updated_at=NOW() WHERE id=$2`, t, id,
	); err != nil {
		return fmt.Errorf("InstanceRepository.UpdateLastBilled: %w", err)
	}
	return nil
}

func (r *InstanceRepository) ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Instance, int, error) {
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM vps.instances WHERE user_id=$1 AND terminated_at IS NULL`, userID); err != nil {
		return nil, 0, err
	}
	var insts []*domain.Instance
	err := r.db.SelectContext(ctx, &insts,
		`SELECT * FROM vps.instances WHERE user_id=$1 AND terminated_at IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	return insts, total, err
}

func (r *InstanceRepository) ListRunning(ctx context.Context) ([]*domain.Instance, error) {
	var insts []*domain.Instance
	err := r.db.SelectContext(ctx, &insts,
		`SELECT * FROM vps.instances WHERE status='running' AND terminated_at IS NULL`)
	return insts, err
}

func (r *InstanceRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE vps.instances SET terminated_at=NOW(), updated_at=NOW() WHERE id=$1`, id)
	return err
}

// ─── SnapshotRepository ───────────────────────────────────────────────────────

type SnapshotRepository struct{ db *sqlx.DB }

func NewSnapshotRepository(db *sqlx.DB) *SnapshotRepository { return &SnapshotRepository{db: db} }

func (r *SnapshotRepository) Create(ctx context.Context, s *domain.Snapshot) error {
	q := `INSERT INTO vps.snapshots (id,instance_id,name,proxmox_name,description,created_at)
		  VALUES (:id,:instance_id,:name,:proxmox_name,:description,:created_at)`
	if _, err := r.db.NamedExecContext(ctx, q, s); err != nil {
		return fmt.Errorf("SnapshotRepository.Create: %w", err)
	}
	return nil
}

func (r *SnapshotRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Snapshot, error) {
	var s domain.Snapshot
	if err := r.db.GetContext(ctx, &s, `SELECT * FROM vps.snapshots WHERE id=$1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSnapshotNotFound
		}
		return nil, fmt.Errorf("SnapshotRepository.GetByID: %w", err)
	}
	return &s, nil
}

func (r *SnapshotRepository) ListByInstance(ctx context.Context, instanceID uuid.UUID) ([]*domain.Snapshot, error) {
	var snaps []*domain.Snapshot
	err := r.db.SelectContext(ctx, &snaps,
		`SELECT * FROM vps.snapshots WHERE instance_id=$1 ORDER BY created_at DESC`, instanceID)
	return snaps, err
}

func (r *SnapshotRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM vps.snapshots WHERE id=$1`, id)
	return err
}
