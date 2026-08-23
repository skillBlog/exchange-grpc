-- +goose Up
CREATE INDEX IF NOT EXISTS idx_markets_allowed_roles_gin ON markets USING GIN (allowed_roles);

-- +goose Down
DROP INDEX IF EXISTS idx_markets_allowed_roles_gin;
