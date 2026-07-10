-- +goose Up
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    market_id TEXT NOT NULL,
    side TEXT NOT NULL,
    price_amount TEXT,
    price_currency TEXT,
    quantity TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id_id ON orders (user_id, id);

-- +goose Down
DROP TABLE IF EXISTS orders;
