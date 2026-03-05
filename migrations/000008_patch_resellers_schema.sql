-- migrate:up
-- Patch 000002: Add missing columns and tables for reseller-service Phase 4
-- Safe to run after initial migration (uses IF NOT EXISTS / ADD COLUMN IF NOT EXISTS)

-- Add missing columns to resellers.accounts
ALTER TABLE resellers.accounts
    ADD COLUMN IF NOT EXISTS email            VARCHAR(255),
    ADD COLUMN IF NOT EXISTS phone            VARCHAR(50),
    ADD COLUMN IF NOT EXISTS address          TEXT,
    ADD COLUMN IF NOT EXISTS tax_id           VARCHAR(100),
    ADD COLUMN IF NOT EXISTS company_name_fix VARCHAR(255),
    ADD COLUMN IF NOT EXISTS commission_pct   DECIMAL(5,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS suspend_reason   TEXT,
    ADD COLUMN IF NOT EXISTS suspended_at     TIMESTAMPTZ;

-- Rename commission_rate → commission_pct if exists (idempotent via DO block)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema='resellers' AND table_name='accounts' AND column_name='commission_rate'
    ) THEN
        ALTER TABLE resellers.accounts RENAME COLUMN commission_rate TO commission_rate_old;
    END IF;
END $$;

-- Add floor_price column to resellers.pricing_overrides (needed by SetPricing validation)
ALTER TABLE resellers.pricing_overrides
    ADD COLUMN IF NOT EXISTS floor_price DECIMAL(12,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Add UNIQUE constraint on (reseller_id, product_id) for ON CONFLICT upsert
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_schema='resellers' AND constraint_name='uq_pricing_reseller_product'
    ) THEN
        ALTER TABLE resellers.pricing_overrides
            ADD CONSTRAINT uq_pricing_reseller_product UNIQUE (reseller_id, product_id);
    END IF;
END $$;

-- API keys table
CREATE TABLE IF NOT EXISTS resellers.api_keys (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reseller_id  UUID NOT NULL REFERENCES resellers.accounts(id) ON DELETE CASCADE,
    name         VARCHAR(255) NOT NULL,
    key_hash     VARCHAR(64) NOT NULL UNIQUE,    -- SHA256(plain_key)
    key_prefix   VARCHAR(20) NOT NULL,           -- first 12 chars for display
    scopes       TEXT[] NOT NULL DEFAULT '{}',
    last_used_at TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Sub-accounts table
CREATE TABLE IF NOT EXISTS resellers.sub_accounts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reseller_id  UUID NOT NULL REFERENCES resellers.accounts(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL UNIQUE,           -- links to users.accounts
    credit_limit DECIMAL(14,4) NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_api_keys_reseller ON resellers.api_keys(reseller_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_hash     ON resellers.api_keys(key_hash) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_sub_accounts_reseller ON resellers.sub_accounts(reseller_id);
CREATE INDEX IF NOT EXISTS idx_sub_accounts_user     ON resellers.sub_accounts(user_id);

-- migrate:down

DROP INDEX IF EXISTS resellers.idx_sub_accounts_user;
DROP INDEX IF EXISTS resellers.idx_sub_accounts_reseller;
DROP INDEX IF EXISTS resellers.idx_api_keys_hash;
DROP INDEX IF EXISTS resellers.idx_api_keys_reseller;
DROP TABLE IF EXISTS resellers.sub_accounts;
DROP TABLE IF EXISTS resellers.api_keys;
