-- migrate:up

CREATE SCHEMA IF NOT EXISTS proxy;

CREATE TABLE proxy.providers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) UNIQUE NOT NULL,  -- 'smartproxy', '711proxy'
    display_name    VARCHAR(255) NOT NULL,
    adapter_type    VARCHAR(50) NOT NULL,
    config          JSONB NOT NULL DEFAULT '{}',  -- encrypted API keys
    is_active       BOOLEAN DEFAULT TRUE,
    priority        INT DEFAULT 0,  -- routing priority (higher = preferred)
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
    provider_order_id VARCHAR(255),           -- ID from provider
    credentials     TEXT,                     -- AES-256-GCM encrypted JSON
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
    order_id    UUID NOT NULL REFERENCES proxy.orders(id) ON DELETE CASCADE,
    event_type  VARCHAR(50) NOT NULL,
    payload     JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Sequence for order numbers
CREATE SEQUENCE IF NOT EXISTS proxy.order_number_seq START 1;

-- Indexes
CREATE INDEX idx_proxy_providers_active ON proxy.providers(is_active, priority DESC);
CREATE INDEX idx_proxy_products_provider ON proxy.products(provider_id);
CREATE INDEX idx_proxy_products_active ON proxy.products(is_active, proxy_type, protocol);
CREATE INDEX idx_proxy_orders_user ON proxy.orders(user_id);
CREATE INDEX idx_proxy_orders_reseller ON proxy.orders(reseller_id);
CREATE INDEX idx_proxy_orders_status ON proxy.orders(status);
CREATE INDEX idx_proxy_orders_expires ON proxy.orders(expires_at) WHERE status = 'active';
CREATE INDEX idx_proxy_orders_number ON proxy.orders(order_number);
CREATE INDEX idx_proxy_order_events_order ON proxy.order_events(order_id, created_at DESC);

-- migrate:down

DROP SEQUENCE IF EXISTS proxy.order_number_seq;

DROP INDEX IF EXISTS idx_proxy_order_events_order;
DROP INDEX IF EXISTS idx_proxy_orders_number;
DROP INDEX IF EXISTS idx_proxy_orders_expires;
DROP INDEX IF EXISTS idx_proxy_orders_status;
DROP INDEX IF EXISTS idx_proxy_orders_reseller;
DROP INDEX IF EXISTS idx_proxy_orders_user;
DROP INDEX IF EXISTS idx_proxy_products_active;
DROP INDEX IF EXISTS idx_proxy_products_provider;
DROP INDEX IF EXISTS idx_proxy_providers_active;

DROP TABLE IF EXISTS proxy.order_events;
DROP TABLE IF EXISTS proxy.orders;
DROP TABLE IF EXISTS proxy.products;
DROP TABLE IF EXISTS proxy.providers;
DROP SCHEMA IF EXISTS proxy;
