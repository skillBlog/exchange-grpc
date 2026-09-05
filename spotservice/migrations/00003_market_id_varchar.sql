-- +goose Up
-- Исторический no-op: 00001 уже создаёт id VARCHAR(64).
-- ALTER COLUMN ... TYPE VARCHAR(64) брал ACCESS EXCLUSIVE на markets без смены типа.
-- Файл оставляем, чтобы не сбивать goose-версию 3 на уже накатанных БД.
SELECT 1;

-- +goose Down
SELECT 1;
