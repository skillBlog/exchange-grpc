-- +goose Up
INSERT INTO markets (id, name, base_asset, quote_asset, enabled, allowed_roles)
VALUES
    ('BTC-USDT', 'Bitcoin / Tether', 'BTC', 'USDT', TRUE, '{}'),
    ('ETH-USDT', 'Ethereum / Tether', 'ETH', 'USDT', TRUE, '{}'),
    ('BNB-USDT', 'BNB / Tether', 'BNB', 'USDT', TRUE, '{trader,admin}'),
    ('SOL-USDT', 'Solana / Tether', 'SOL', 'USDT', FALSE, '{}'),
    ('XRP-USDT', 'Ripple / Tether', 'XRP', 'USDT', FALSE, '{}')
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM markets
WHERE id IN ('BTC-USDT', 'ETH-USDT', 'BNB-USDT', 'SOL-USDT', 'XRP-USDT');
