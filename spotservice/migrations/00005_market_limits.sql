-- +goose Up
ALTER TABLE markets ADD COLUMN IF NOT EXISTS min_order_size NUMERIC NOT NULL DEFAULT 0.00000001;
ALTER TABLE markets ADD COLUMN IF NOT EXISTS quantity_precision INTEGER NOT NULL DEFAULT 8;
ALTER TABLE markets ADD COLUMN IF NOT EXISTS min_notional NUMERIC NOT NULL DEFAULT 10;

UPDATE markets SET min_order_size = 0.0001, quantity_precision = 8, min_notional = 10 WHERE id = 'BTC-USDT';
UPDATE markets SET min_order_size = 0.001, quantity_precision = 8, min_notional = 10 WHERE id = 'ETH-USDT';
UPDATE markets SET min_order_size = 0.01, quantity_precision = 6, min_notional = 10 WHERE id = 'BNB-USDT';
UPDATE markets SET min_order_size = 0.01, quantity_precision = 4, min_notional = 10 WHERE id = 'SOL-USDT';
UPDATE markets SET min_order_size = 1, quantity_precision = 2, min_notional = 10 WHERE id = 'XRP-USDT';

-- +goose Down
ALTER TABLE markets DROP COLUMN IF EXISTS min_order_size;
ALTER TABLE markets DROP COLUMN IF EXISTS quantity_precision;
ALTER TABLE markets DROP COLUMN IF EXISTS min_notional;
