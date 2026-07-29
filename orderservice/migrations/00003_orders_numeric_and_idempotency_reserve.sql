-- +goose Up
-- Для уже применённых миграций: reserve-before-create + NUMERIC money/quantity.

ALTER TABLE idempotency_keys
    DROP CONSTRAINT IF EXISTS idempotency_keys_order_id_fkey;

ALTER TABLE orders
    ALTER COLUMN price_amount TYPE NUMERIC
        USING NULLIF(TRIM(price_amount::text), '')::NUMERIC,
    ALTER COLUMN quantity TYPE NUMERIC
        USING TRIM(quantity::text)::NUMERIC;

-- +goose Down
ALTER TABLE orders
    ALTER COLUMN price_amount TYPE TEXT USING price_amount::text,
    ALTER COLUMN quantity TYPE TEXT USING quantity::text;

ALTER TABLE idempotency_keys
    ADD CONSTRAINT idempotency_keys_order_id_fkey
        FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE;
