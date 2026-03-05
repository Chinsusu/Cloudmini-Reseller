-- migrate:up

CREATE SCHEMA IF NOT EXISTS notifications;

CREATE TABLE notifications.in_app (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    title           VARCHAR(255) NOT NULL,
    body            TEXT NOT NULL,
    type            VARCHAR(50) NOT NULL,  -- order|billing|system|vps
    reference_type  VARCHAR(50),
    reference_id    UUID,
    is_read         BOOLEAN DEFAULT FALSE,
    read_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE notifications.delivery_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id        UUID,              -- original NATS event ID if tracked
    channel         VARCHAR(20) NOT NULL,  -- email|webhook|in_app
    recipient       VARCHAR(500) NOT NULL, -- email addr or webhook URL
    template        VARCHAR(100),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    -- pending|sent|failed|retrying
    attempts        INT NOT NULL DEFAULT 0,
    last_error      TEXT,
    sent_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_notifications_user ON notifications.in_app(user_id, created_at DESC);
CREATE INDEX idx_notifications_unread ON notifications.in_app(user_id) WHERE is_read = FALSE;
CREATE INDEX idx_delivery_log_status ON notifications.delivery_log(status, created_at DESC);
CREATE INDEX idx_delivery_log_channel ON notifications.delivery_log(channel, created_at DESC);

-- migrate:down

DROP INDEX IF EXISTS idx_delivery_log_channel;
DROP INDEX IF EXISTS idx_delivery_log_status;
DROP INDEX IF EXISTS idx_notifications_unread;
DROP INDEX IF EXISTS idx_notifications_user;

DROP TABLE IF EXISTS notifications.delivery_log;
DROP TABLE IF EXISTS notifications.in_app;
DROP SCHEMA IF EXISTS notifications;
