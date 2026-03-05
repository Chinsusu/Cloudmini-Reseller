-- migrate:up

CREATE SCHEMA IF NOT EXISTS logs;

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

-- Create initial partitions (auto-provision new ones via cron)
CREATE TABLE logs.entries_2026_03 PARTITION OF logs.entries
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE logs.entries_2026_04 PARTITION OF logs.entries
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE TABLE logs.entries_2026_05 PARTITION OF logs.entries
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE logs.entries_2026_06 PARTITION OF logs.entries
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE logs.entries_2026_07 PARTITION OF logs.entries
    FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
CREATE TABLE logs.entries_2026_08 PARTITION OF logs.entries
    FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');
CREATE TABLE logs.entries_2026_09 PARTITION OF logs.entries
    FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');
CREATE TABLE logs.entries_2026_10 PARTITION OF logs.entries
    FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');
CREATE TABLE logs.entries_2026_11 PARTITION OF logs.entries
    FOR VALUES FROM ('2026-11-01') TO ('2026-12-01');
CREATE TABLE logs.entries_2026_12 PARTITION OF logs.entries
    FOR VALUES FROM ('2026-12-01') TO ('2027-01-01');
CREATE TABLE logs.entries_2027_01 PARTITION OF logs.entries
    FOR VALUES FROM ('2027-01-01') TO ('2027-02-01');
CREATE TABLE logs.entries_2027_02 PARTITION OF logs.entries
    FOR VALUES FROM ('2027-02-01') TO ('2027-03-01');

-- Indexes (applied on each partition automatically due to inheritance)
CREATE INDEX idx_logs_user ON logs.entries(user_id, created_at DESC);
CREATE INDEX idx_logs_resource ON logs.entries(resource_type, resource_id);
CREATE INDEX idx_logs_action ON logs.entries(action, created_at DESC);
CREATE INDEX idx_logs_level ON logs.entries(level, created_at DESC) WHERE level IN ('WARN', 'ERROR');
CREATE INDEX idx_logs_request ON logs.entries(request_id);
CREATE INDEX idx_logs_service ON logs.entries(service_name, created_at DESC);

-- migrate:down

DROP INDEX IF EXISTS idx_logs_service;
DROP INDEX IF EXISTS idx_logs_request;
DROP INDEX IF EXISTS idx_logs_level;
DROP INDEX IF EXISTS idx_logs_action;
DROP INDEX IF EXISTS idx_logs_resource;
DROP INDEX IF EXISTS idx_logs_user;

DROP TABLE IF EXISTS logs.entries_2027_02;
DROP TABLE IF EXISTS logs.entries_2027_01;
DROP TABLE IF EXISTS logs.entries_2026_12;
DROP TABLE IF EXISTS logs.entries_2026_11;
DROP TABLE IF EXISTS logs.entries_2026_10;
DROP TABLE IF EXISTS logs.entries_2026_09;
DROP TABLE IF EXISTS logs.entries_2026_08;
DROP TABLE IF EXISTS logs.entries_2026_07;
DROP TABLE IF EXISTS logs.entries_2026_06;
DROP TABLE IF EXISTS logs.entries_2026_05;
DROP TABLE IF EXISTS logs.entries_2026_04;
DROP TABLE IF EXISTS logs.entries_2026_03;
DROP TABLE IF EXISTS logs.entries;
DROP SCHEMA IF EXISTS logs;
