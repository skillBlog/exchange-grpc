-- +goose Up
CREATE TABLE IF NOT EXISTS idempotency_keys (
    user_id UUID NOT NULL,
    idempotency_key TEXT NOT NULL,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, idempotency_key)
);

-- +goose Down
DROP TABLE IF EXISTS idempotency_keys;
