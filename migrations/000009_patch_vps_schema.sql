-- migrate:up
-- Patch 000009: Align vps schema column names with Go code (vps-service Phase 3)
-- Adds missing cols: node_name, idempotency_key updates node col names

-- vps.nodes: add missing columns referenced by nodeRepo
ALTER TABLE vps.nodes
    ADD COLUMN IF NOT EXISTS display_name    VARCHAR(255),
    ADD COLUMN IF NOT EXISTS proxmox_host    VARCHAR(255),
    ADD COLUMN IF NOT EXISTS proxmox_port    INT NOT NULL DEFAULT 8006,
    ADD COLUMN IF NOT EXISTS total_ram_mb_2  BIGINT;

-- Rename host → proxmox_host if column exists as 'host'
-- (vps-service code uses proxmox_host, original migration uses host)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema='vps' AND table_name='nodes' AND column_name='host'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema='vps' AND table_name='nodes' AND column_name='proxmox_host'
    ) THEN
        ALTER TABLE vps.nodes RENAME COLUMN host TO proxmox_host;
    END IF;
END $$;

-- vps.instances: align columns with Go vps-service domain
-- Go code uses: vmid (not proxmox_vmid), cpu (not cpu_cores), node_name (string beside node_id)
ALTER TABLE vps.instances
    ADD COLUMN IF NOT EXISTS node_name       VARCHAR(100),
    ADD COLUMN IF NOT EXISTS ip_address_str  VARCHAR(50);   -- fallback VARCHAR for ip_address

-- Add vmid column (alias proxmox_vmid → vmid)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema='vps' AND table_name='instances' AND column_name='proxmox_vmid'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema='vps' AND table_name='instances' AND column_name='vmid'
    ) THEN
        ALTER TABLE vps.instances RENAME COLUMN proxmox_vmid TO vmid;
    END IF;
END $$;

-- Add cpu column if only cpu_cores exists
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema='vps' AND table_name='instances' AND column_name='cpu'
    ) THEN
        ALTER TABLE vps.instances ADD COLUMN cpu INT;
        UPDATE vps.instances SET cpu = cpu_cores WHERE cpu IS NULL;
    END IF;
END $$;

-- vps.snapshots: add description column (Go code stores description)
ALTER TABLE vps.snapshots
    ADD COLUMN IF NOT EXISTS description    TEXT,
    ADD COLUMN IF NOT EXISTS proxmox_name   VARCHAR(100);   -- aliased proxmox_snapname

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema='vps' AND table_name='snapshots' AND column_name='proxmox_snapname'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema='vps' AND table_name='snapshots' AND column_name='proxmox_name'
    ) THEN
        ALTER TABLE vps.snapshots RENAME COLUMN proxmox_snapname TO proxmox_name;
    END IF;
END $$;

-- Add size_gb column (Go uses size_gb not size_mb)
ALTER TABLE vps.snapshots
    ADD COLUMN IF NOT EXISTS size_gb DECIMAL(10,3);

-- Additional index for billing cron (ListRunning)
CREATE INDEX IF NOT EXISTS idx_vps_instances_billing
    ON vps.instances(last_billed_at) WHERE status = 'running' AND terminated_at IS NULL;

-- migrate:down
DROP INDEX IF EXISTS vps.idx_vps_instances_billing;
