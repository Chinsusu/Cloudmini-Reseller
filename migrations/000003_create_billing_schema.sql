-- migrate:up

CREATE SCHEMA IF NOT EXISTS billing;

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

-- Sequence for transaction numbers
CREATE SEQUENCE IF NOT EXISTS billing.txn_number_seq START 1;
CREATE SEQUENCE IF NOT EXISTS billing.payment_number_seq START 1;

-- Indexes
CREATE INDEX idx_wallets_user ON billing.wallets(user_id);
CREATE INDEX idx_transactions_wallet ON billing.transactions(wallet_id);
CREATE INDEX idx_transactions_user ON billing.transactions(user_id);
CREATE INDEX idx_transactions_type ON billing.transactions(type, created_at DESC);
CREATE INDEX idx_transactions_reference ON billing.transactions(reference_type, reference_id);
CREATE INDEX idx_payments_user ON billing.payments(user_id);
CREATE INDEX idx_payments_status ON billing.payments(status);
CREATE INDEX idx_pricing_rules_reseller ON billing.pricing_rules(reseller_id);
CREATE INDEX idx_metering_instance ON billing.vps_metering(instance_id);
CREATE INDEX idx_metering_period ON billing.vps_metering(period_start, period_end);

-- migrate:down

DROP SEQUENCE IF EXISTS billing.payment_number_seq;
DROP SEQUENCE IF EXISTS billing.txn_number_seq;

DROP INDEX IF EXISTS idx_metering_period;
DROP INDEX IF EXISTS idx_metering_instance;
DROP INDEX IF EXISTS idx_pricing_rules_reseller;
DROP INDEX IF EXISTS idx_payments_status;
DROP INDEX IF EXISTS idx_payments_user;
DROP INDEX IF EXISTS idx_transactions_reference;
DROP INDEX IF EXISTS idx_transactions_type;
DROP INDEX IF EXISTS idx_transactions_user;
DROP INDEX IF EXISTS idx_transactions_wallet;
DROP INDEX IF EXISTS idx_wallets_user;

DROP TABLE IF EXISTS billing.vps_metering;
DROP TABLE IF EXISTS billing.pricing_rules;
DROP TABLE IF EXISTS billing.payments;
DROP TABLE IF EXISTS billing.transactions;
DROP TABLE IF EXISTS billing.wallets;
DROP SCHEMA IF EXISTS billing;
