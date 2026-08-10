-- +goose Up
-- Статусная модель idempotency: reserved → completed|failed + TTL для перезаписи.

ALTER TABLE idempotency_keys
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'completed',
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '24 hours');

UPDATE idempotency_keys
SET status = 'completed'
WHERE status IS NULL OR status = '';

ALTER TABLE idempotency_keys
    DROP CONSTRAINT IF EXISTS idempotency_keys_status_check;

ALTER TABLE idempotency_keys
    ADD CONSTRAINT idempotency_keys_status_check
        CHECK (status IN ('reserved', 'completed', 'failed'));

CREATE INDEX IF NOT EXISTS idempotency_keys_expires_at_idx
    ON idempotency_keys (expires_at);

-- +goose Down
DROP INDEX IF EXISTS idempotency_keys_expires_at_idx;

ALTER TABLE idempotency_keys
    DROP CONSTRAINT IF EXISTS idempotency_keys_status_check;

ALTER TABLE idempotency_keys
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS status;
