-- migrate:up

CREATE SCHEMA IF NOT EXISTS vps;

CREATE TABLE vps.nodes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) UNIQUE NOT NULL,   -- 'node-01', 'dedicated-hcm-01'
    host            VARCHAR(255) NOT NULL,           -- Proxmox API URL
    node_id         VARCHAR(100) NOT NULL,           -- Proxmox node name
    location        VARCHAR(100) NOT NULL,
    node_type       VARCHAR(20) NOT NULL DEFAULT 'self', -- self|dedicated
    total_ram_mb    BIGINT NOT NULL,
    total_cpu       INT NOT NULL,
    total_disk_gb   BIGINT NOT NULL,
    reserved_ram_mb BIGINT NOT NULL DEFAULT 0,
    reserved_cpu    INT NOT NULL DEFAULT 0,
    reserved_disk_gb BIGINT NOT NULL DEFAULT 0,
    status          VARCHAR(20) DEFAULT 'online',    -- online|offline|maintenance
    last_health_at  TIMESTAMPTZ,
    priority        INT DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE vps.plans (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL,
    slug            VARCHAR(50) UNIQUE NOT NULL,  -- 'starter-1', 'pro-2'
    cpu_cores       INT NOT NULL,
    ram_mb          INT NOT NULL,
    disk_gb         INT NOT NULL,
    bandwidth_gb    INT,                          -- NULL = unlimited
    os_templates    TEXT[] NOT NULL DEFAULT '{}',
    hourly_rate     DECIMAL(10,6) NOT NULL,
    monthly_rate    DECIMAL(10,4),               -- pre-calculated for display
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE vps.instances (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_number VARCHAR(30) UNIQUE NOT NULL,  -- VS-2025-000001
    user_id         UUID NOT NULL,
    reseller_id     UUID,
    plan_id         UUID NOT NULL REFERENCES vps.plans(id),
    node_id         UUID NOT NULL REFERENCES vps.nodes(id),
    proxmox_vmid    INT,                          -- VM ID in Proxmox
    hostname        VARCHAR(255) NOT NULL,
    os_template     VARCHAR(100) NOT NULL,
    status          VARCHAR(30) NOT NULL DEFAULT 'pending',
    -- pending|provisioning|booting|running|suspended|stopped|terminated|failed
    ip_address      INET,
    ipv6_address    INET,
    ssh_port        INT DEFAULT 22,
    root_password   TEXT,                         -- AES-256-GCM encrypted
    cpu_cores       INT NOT NULL,
    ram_mb          INT NOT NULL,
    disk_gb         INT NOT NULL,
    billing_type    VARCHAR(20) NOT NULL DEFAULT 'hourly',
    billing_started_at TIMESTAMPTZ,
    last_billed_at  TIMESTAMPTZ,
    suspended_at    TIMESTAMPTZ,
    terminated_at   TIMESTAMPTZ,
    idempotency_key VARCHAR(255) UNIQUE,
    request_id      UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE vps.instance_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id     UUID NOT NULL REFERENCES vps.instances(id) ON DELETE CASCADE,
    event_type      VARCHAR(50) NOT NULL,
    -- provision_started|booting|ready|suspended|resumed|snapshot_created|terminated
    from_status     VARCHAR(30),
    to_status       VARCHAR(30),
    triggered_by    VARCHAR(20) NOT NULL,  -- user|system|admin
    payload         JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE vps.snapshots (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id     UUID NOT NULL REFERENCES vps.instances(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    proxmox_snapname VARCHAR(100) NOT NULL,
    size_mb         BIGINT,
    status          VARCHAR(20) DEFAULT 'creating',  -- creating|ready|deleting|failed
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Sequence for instance numbers
CREATE SEQUENCE IF NOT EXISTS vps.instance_number_seq START 1;

-- Indexes
CREATE INDEX idx_vps_nodes_status ON vps.nodes(status, priority DESC);
CREATE INDEX idx_vps_plans_active ON vps.plans(is_active);
CREATE INDEX idx_vps_instances_user ON vps.instances(user_id);
CREATE INDEX idx_vps_instances_reseller ON vps.instances(reseller_id);
CREATE INDEX idx_vps_instances_status ON vps.instances(status);
CREATE INDEX idx_vps_instances_node ON vps.instances(node_id);
CREATE INDEX idx_vps_instances_running ON vps.instances(node_id, last_billed_at) WHERE status = 'running';
CREATE INDEX idx_vps_events_instance ON vps.instance_events(instance_id, created_at DESC);
CREATE INDEX idx_vps_snapshots_instance ON vps.snapshots(instance_id);

-- migrate:down

DROP SEQUENCE IF EXISTS vps.instance_number_seq;

DROP INDEX IF EXISTS idx_vps_snapshots_instance;
DROP INDEX IF EXISTS idx_vps_events_instance;
DROP INDEX IF EXISTS idx_vps_instances_running;
DROP INDEX IF EXISTS idx_vps_instances_node;
DROP INDEX IF EXISTS idx_vps_instances_status;
DROP INDEX IF EXISTS idx_vps_instances_reseller;
DROP INDEX IF EXISTS idx_vps_instances_user;
DROP INDEX IF EXISTS idx_vps_plans_active;
DROP INDEX IF EXISTS idx_vps_nodes_status;

DROP TABLE IF EXISTS vps.snapshots;
DROP TABLE IF EXISTS vps.instance_events;
DROP TABLE IF EXISTS vps.instances;
DROP TABLE IF EXISTS vps.plans;
DROP TABLE IF EXISTS vps.nodes;
DROP SCHEMA IF EXISTS vps;
