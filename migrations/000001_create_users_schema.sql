-- migrate:up

CREATE SCHEMA IF NOT EXISTS users;

CREATE TABLE users.accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) UNIQUE NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    full_name       VARCHAR(255) NOT NULL,
    phone           VARCHAR(20),
    role            VARCHAR(20) NOT NULL DEFAULT 'user',  -- user|reseller|admin|super_admin
    status          VARCHAR(20) NOT NULL DEFAULT 'active', -- active|suspended|banned|pending_verification
    reseller_id     UUID,  -- references resellers.accounts(id)
    email_verified  BOOLEAN DEFAULT FALSE,
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ  -- soft delete
);

CREATE TABLE users.sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users.accounts(id) ON DELETE CASCADE,
    refresh_token   VARCHAR(512) UNIQUE NOT NULL,
    ip_address      INET,
    user_agent      TEXT,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at      TIMESTAMPTZ
);

CREATE TABLE users.api_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users.accounts(id) ON DELETE CASCADE,
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
CREATE INDEX idx_users_status ON users.accounts(status);
CREATE INDEX idx_sessions_user ON users.sessions(user_id);
CREATE INDEX idx_sessions_token ON users.sessions(refresh_token);
CREATE INDEX idx_sessions_expires ON users.sessions(expires_at);
CREATE INDEX idx_api_keys_user ON users.api_keys(user_id);
CREATE INDEX idx_api_keys_hash ON users.api_keys(key_hash);

-- migrate:down

DROP INDEX IF EXISTS idx_api_keys_hash;
DROP INDEX IF EXISTS idx_api_keys_user;
DROP INDEX IF EXISTS idx_sessions_expires;
DROP INDEX IF EXISTS idx_sessions_token;
DROP INDEX IF EXISTS idx_sessions_user;
DROP INDEX IF EXISTS idx_users_status;
DROP INDEX IF EXISTS idx_users_reseller;
DROP INDEX IF EXISTS idx_users_email;

DROP TABLE IF EXISTS users.api_keys;
DROP TABLE IF EXISTS users.sessions;
DROP TABLE IF EXISTS users.accounts;
DROP SCHEMA IF EXISTS users;
