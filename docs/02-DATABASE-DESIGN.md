# Database Design — ProxyVPS Platform

**Document ID**: PVP-DOC-002  
**Version**: 1.0.0  
**Status**: Approved  
**Last Updated**: 2025-01-01  

---

## 1. Overview

Mỗi service có schema riêng trong cùng một PostgreSQL instance (dev) hoặc PostgreSQL instance riêng (production). Dùng schema-per-service để tách biệt logical mà không cần multiple databases.

```
PostgreSQL Instance
├── schema: users
├── schema: proxy
├── schema: vps
├── schema: billing
├── schema: logs
├── schema: notifications
└── schema: resellers
```

---

## 2. Schema: Users

```sql
-- schema: users

CREATE TABLE users.accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) UNIQUE NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    full_name       VARCHAR(255) NOT NULL,
    phone           VARCHAR(20),
    role            VARCHAR(20) NOT NULL DEFAULT 'user',  -- user|reseller|admin|super_admin
    status          VARCHAR(20) NOT NULL DEFAULT 'active', -- active|suspended|banned
    reseller_id     UUID REFERENCES resellers.accounts(id) ON DELETE SET NULL,
    email_verified  BOOLEAN DEFAULT FALSE,
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ  -- soft delete
);

CREATE TABLE users.sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users.accounts(id),
    refresh_token   VARCHAR(512) UNIQUE NOT NULL,
    ip_address      INET,
    user_agent      TEXT,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at      TIMESTAMPTZ
);

CREATE TABLE users.api_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users.accounts(id),
    name            VARCHAR(100) NOT NULL,
    key_hash        VARCHAR(255) UNIQUE NOT NULL,  -- SHA256 of actual key
    key_prefix      VARCHAR(10) NOT NULL,          -- first 8 chars for display
    scopes          TEXT[] NOT NULL DEFAULT '{}',  -- ['read', 'write', 'proxy', 'vps']
    last_used_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at      TIMESTAMPTZ
);

-- Indexes
CREATE INDEX idx_users_email ON users.accounts(email);
CREATE INDEX idx_users_reseller ON users.accounts(reseller_id);
CREATE INDEX idx_sessions_user ON users.sessions(user_id);
CREATE INDEX idx_sessions_token ON users.sessions(refresh_token);
```

---

## 3. Schema: Proxy

```sql
-- schema: proxy

CREATE TABLE proxy.providers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) UNIQUE NOT NULL,  -- 'smartproxy', '711proxy'
    display_name    VARCHAR(255) NOT NULL,
    adapter_type    VARCHAR(50) NOT NULL,
    config          JSONB NOT NULL DEFAULT '{}',  -- encrypted API keys in vault
    is_active       BOOLEAN DEFAULT TRUE,
    priority        INT DEFAULT 0,  -- routing priority
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE proxy.products (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id     UUID NOT NULL REFERENCES proxy.providers(id),
    name            VARCHAR(255) NOT NULL,
    proxy_type      VARCHAR(50) NOT NULL,   -- residential|datacenter|mobile|isp
    protocol        VARCHAR(20) NOT NULL,   -- http|socks5|https
    location        VARCHAR(100),           -- country/city
    duration_days   INT,                    -- NULL = bandwidth-based
    bandwidth_gb    DECIMAL(10,2),          -- NULL = time-based
    base_cost       DECIMAL(12,4) NOT NULL, -- cost from provider
    is_active       BOOLEAN DEFAULT TRUE,
    metadata        JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE proxy.orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_number    VARCHAR(30) UNIQUE NOT NULL,  -- PX-2025-000001
    user_id         UUID NOT NULL,
    reseller_id     UUID,
    product_id      UUID NOT NULL REFERENCES proxy.products(id),
    provider_id     UUID NOT NULL REFERENCES proxy.providers(id),
    status          VARCHAR(30) NOT NULL DEFAULT 'pending',
    -- pending|processing|active|expired|cancelled|failed|refunded
    quantity        INT NOT NULL DEFAULT 1,
    unit_price      DECIMAL(12,4) NOT NULL,   -- price charged to user
    total_amount    DECIMAL(12,4) NOT NULL,
    provider_order_id VARCHAR(255),           -- ID từ provider
    credentials     JSONB,                    -- encrypted: {host, port, username, password}
    activated_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    cancelled_at    TIMESTAMPTZ,
    cancel_reason   TEXT,
    idempotency_key VARCHAR(255) UNIQUE,
    request_id      UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE proxy.order_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID NOT NULL REFERENCES proxy.orders(id),
    event_type  VARCHAR(50) NOT NULL,
    payload     JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_proxy_orders_user ON proxy.orders(user_id);
CREATE INDEX idx_proxy_orders_status ON proxy.orders(status);
CREATE INDEX idx_proxy_orders_expires ON proxy.orders(expires_at) WHERE status = 'active';
CREATE INDEX idx_proxy_orders_number ON proxy.orders(order_number);
```

---

## 4. Schema: VPS

```sql
-- schema: vps

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
    root_password   TEXT,                         -- encrypted
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
    instance_id     UUID NOT NULL REFERENCES vps.instances(id),
    event_type      VARCHAR(50) NOT NULL,
    -- provision_started|booting|ready|suspended|resumed|snapshot_created...
    from_status     VARCHAR(30),
    to_status       VARCHAR(30),
    triggered_by    VARCHAR(20) NOT NULL,  -- user|system|admin
    payload         JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE vps.snapshots (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id     UUID NOT NULL REFERENCES vps.instances(id),
    name            VARCHAR(255) NOT NULL,
    proxmox_snapname VARCHAR(100) NOT NULL,
    size_mb         BIGINT,
    status          VARCHAR(20) DEFAULT 'creating',  -- creating|ready|deleting|failed
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_vps_instances_user ON vps.instances(user_id);
CREATE INDEX idx_vps_instances_status ON vps.instances(status);
CREATE INDEX idx_vps_instances_node ON vps.instances(node_id);
CREATE INDEX idx_vps_instances_running ON vps.instances(node_id) WHERE status = 'running';
```

---

## 5. Schema: Billing

```sql
-- schema: billing

CREATE TABLE billing.wallets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID UNIQUE NOT NULL,
    balance         DECIMAL(14,4) NOT NULL DEFAULT 0,
    hold_amount     DECIMAL(14,4) NOT NULL DEFAULT 0,  -- reserved for pending ops
    currency        VARCHAR(3) NOT NULL DEFAULT 'USD',
    low_balance_threshold DECIMAL(14,4) DEFAULT 5.00,
    last_alert_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT positive_balance CHECK (balance >= 0),
    CONSTRAINT positive_hold CHECK (hold_amount >= 0)
);

CREATE TABLE billing.transactions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    txn_number      VARCHAR(30) UNIQUE NOT NULL,  -- TXN-2025-000001
    wallet_id       UUID NOT NULL REFERENCES billing.wallets(id),
    user_id         UUID NOT NULL,
    type            VARCHAR(30) NOT NULL,
    -- deposit|withdrawal|order_charge|order_refund|hold|hold_release|adjustment
    amount          DECIMAL(14,4) NOT NULL,
    balance_before  DECIMAL(14,4) NOT NULL,
    balance_after   DECIMAL(14,4) NOT NULL,
    reference_type  VARCHAR(50),    -- proxy_order|vps_instance|payment
    reference_id    UUID,
    description     TEXT,
    metadata        JSONB DEFAULT '{}',
    request_id      UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE billing.payments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_number  VARCHAR(30) UNIQUE NOT NULL,  -- PAY-2025-000001
    user_id         UUID NOT NULL,
    wallet_id       UUID NOT NULL REFERENCES billing.wallets(id),
    gateway         VARCHAR(30) NOT NULL,  -- stripe|vnpay|momo
    gateway_txn_id  VARCHAR(255),
    amount          DECIMAL(14,4) NOT NULL,
    currency        VARCHAR(3) NOT NULL DEFAULT 'USD',
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    -- pending|processing|completed|failed|refunded
    gateway_response JSONB DEFAULT '{}',
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE billing.pricing_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reseller_id     UUID,          -- NULL = global pricing
    product_type    VARCHAR(20) NOT NULL,  -- proxy|vps
    product_id      UUID,          -- NULL = applies to all products of type
    markup_type     VARCHAR(20) NOT NULL DEFAULT 'percentage',  -- percentage|fixed
    markup_value    DECIMAL(10,4) NOT NULL,
    min_price       DECIMAL(12,4),
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE billing.vps_metering (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id     UUID NOT NULL,
    period_start    TIMESTAMPTZ NOT NULL,
    period_end      TIMESTAMPTZ NOT NULL,
    hours_billed    DECIMAL(10,4) NOT NULL,
    rate            DECIMAL(10,6) NOT NULL,
    amount          DECIMAL(14,4) NOT NULL,
    txn_id          UUID REFERENCES billing.transactions(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_transactions_wallet ON billing.transactions(wallet_id);
CREATE INDEX idx_transactions_user ON billing.transactions(user_id);
CREATE INDEX idx_transactions_reference ON billing.transactions(reference_type, reference_id);
CREATE INDEX idx_payments_user ON billing.payments(user_id);
CREATE INDEX idx_metering_instance ON billing.vps_metering(instance_id);
CREATE INDEX idx_metering_period ON billing.vps_metering(period_start, period_end);
```

---

## 6. Schema: Logs

```sql
-- schema: logs

-- Partitioned by month for performance
CREATE TABLE logs.entries (
    id              UUID NOT NULL DEFAULT gen_random_uuid(),
    request_id      UUID,
    trace_id        UUID,
    service_name    VARCHAR(50) NOT NULL,
    user_id         UUID,
    reseller_id     UUID,
    actor_type      VARCHAR(20) NOT NULL,  -- user|system|admin|reseller|cron
    actor_id        UUID,
    action          VARCHAR(100) NOT NULL,
    level           VARCHAR(10) NOT NULL DEFAULT 'INFO',  -- DEBUG|INFO|WARN|ERROR
    resource_type   VARCHAR(50),      -- proxy_order|vps_instance|wallet|...
    resource_id     UUID,
    message         TEXT NOT NULL,
    payload         JSONB DEFAULT '{}',
    duration_ms     INT,
    ip_address      INET,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- Create monthly partitions (auto-create via pg_partman or cron)
CREATE TABLE logs.entries_2025_01 PARTITION OF logs.entries
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
CREATE TABLE logs.entries_2025_02 PARTITION OF logs.entries
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
-- ... auto-provision monthly

-- Indexes (per partition)
CREATE INDEX idx_logs_user ON logs.entries(user_id, created_at DESC);
CREATE INDEX idx_logs_resource ON logs.entries(resource_type, resource_id);
CREATE INDEX idx_logs_action ON logs.entries(action, created_at DESC);
CREATE INDEX idx_logs_level ON logs.entries(level) WHERE level IN ('WARN', 'ERROR');
CREATE INDEX idx_logs_request ON logs.entries(request_id);
```

---

## 7. Schema: Resellers

```sql
-- schema: resellers

CREATE TABLE resellers.accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID UNIQUE NOT NULL,   -- links to users.accounts
    company_name    VARCHAR(255),
    slug            VARCHAR(100) UNIQUE,    -- for white-label subdomain later
    status          VARCHAR(20) DEFAULT 'active',
    wallet_id       UUID UNIQUE,            -- reseller has own wallet
    commission_rate DECIMAL(5,4) DEFAULT 0, -- future use
    credit_limit    DECIMAL(14,4) DEFAULT 0, -- max user credit under this reseller
    allow_api       BOOLEAN DEFAULT TRUE,
    allow_whitelabel BOOLEAN DEFAULT FALSE,
    notes           TEXT,
    approved_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE resellers.pricing_overrides (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reseller_id     UUID NOT NULL REFERENCES resellers.accounts(id),
    product_type    VARCHAR(20) NOT NULL,
    product_id      UUID NOT NULL,
    sell_price      DECIMAL(12,4) NOT NULL,  -- price reseller charges their users
    cost_price      DECIMAL(12,4) NOT NULL,  -- price reseller pays (platform price)
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 8. Indexing Strategy

### 8.1 Principles
- Index mọi foreign key
- Index columns thường xuất hiện trong WHERE + ORDER BY
- Partial index cho các bộ lọc phổ biến (status = 'active', status = 'running')
- Không over-index: mỗi index có write cost

### 8.2 Query Patterns Ưu Tiên
```sql
-- Most frequent queries (must be < 5ms)
SELECT * FROM proxy.orders WHERE user_id = ? AND status = 'active';
SELECT balance FROM billing.wallets WHERE user_id = ?;
SELECT * FROM logs.entries WHERE user_id = ? ORDER BY created_at DESC LIMIT 50;
SELECT * FROM vps.instances WHERE node_id = ? AND status = 'running';
```

---

## 9. Data Retention Policy

| Data | Retention | Action |
|---|---|---|
| Log entries | 12 months hot, 24 months cold | Move to archive partition |
| Transaction history | Permanent | Never delete |
| Session tokens | 7 days | Expired sessions deleted by cron |
| Order events | 6 months | Aggregated then archived |
| Deleted users | 90 days | Soft delete → hard delete |
