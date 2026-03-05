-- migrate:up

CREATE SCHEMA IF NOT EXISTS resellers;

CREATE TABLE resellers.accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID UNIQUE NOT NULL,   -- links to users.accounts
    company_name    VARCHAR(255),
    slug            VARCHAR(100) UNIQUE,    -- for white-label subdomain later
    status          VARCHAR(20) DEFAULT 'active',    -- active|suspended|pending
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
    reseller_id     UUID NOT NULL REFERENCES resellers.accounts(id) ON DELETE CASCADE,
    product_type    VARCHAR(20) NOT NULL,   -- proxy|vps
    product_id      UUID NOT NULL,
    sell_price      DECIMAL(12,4) NOT NULL, -- price reseller charges their users
    cost_price      DECIMAL(12,4) NOT NULL, -- price reseller pays (platform price)
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE resellers.webhooks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reseller_id     UUID NOT NULL REFERENCES resellers.accounts(id) ON DELETE CASCADE,
    url             VARCHAR(500) NOT NULL,
    secret          VARCHAR(255) NOT NULL,  -- HMAC signing key (encrypted)
    events          TEXT[] NOT NULL DEFAULT '{}',  -- event filter, empty = all
    is_active       BOOLEAN DEFAULT TRUE,
    failure_count   INT DEFAULT 0,
    last_success_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_resellers_user ON resellers.accounts(user_id);
CREATE INDEX idx_resellers_status ON resellers.accounts(status);
CREATE INDEX idx_pricing_reseller ON resellers.pricing_overrides(reseller_id);
CREATE INDEX idx_pricing_product ON resellers.pricing_overrides(product_type, product_id);
CREATE INDEX idx_webhooks_reseller ON resellers.webhooks(reseller_id);

-- Now add FK from users.accounts.reseller_id → resellers.accounts(id)
-- (Deferred because users schema was created first)
ALTER TABLE users.accounts
    ADD CONSTRAINT fk_users_reseller
    FOREIGN KEY (reseller_id) REFERENCES resellers.accounts(id) ON DELETE SET NULL;

-- migrate:down

ALTER TABLE users.accounts DROP CONSTRAINT IF EXISTS fk_users_reseller;

DROP INDEX IF EXISTS idx_webhooks_reseller;
DROP INDEX IF EXISTS idx_pricing_product;
DROP INDEX IF EXISTS idx_pricing_reseller;
DROP INDEX IF EXISTS idx_resellers_status;
DROP INDEX IF EXISTS idx_resellers_user;

DROP TABLE IF EXISTS resellers.webhooks;
DROP TABLE IF EXISTS resellers.pricing_overrides;
DROP TABLE IF EXISTS resellers.accounts;
DROP SCHEMA IF EXISTS resellers;
