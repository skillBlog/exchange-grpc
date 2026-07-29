-- +goose Up
-- order_id без FK: ключ резервируется до INSERT в orders (антидублирование важнее orphan key).
CREATE TABLE IF NOT EXISTS idempotency_keys (
    user_id UUID NOT NULL,
    idempotency_key TEXT NOT NULL,
    order_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, idempotency_key)
);

-- +goose Down
DROP TABLE IF EXISTS idempotency_keys;
