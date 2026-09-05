-- +goose Up
-- Историческая миграция: в ранних ревизиях у idempotency_keys был FK на orders,
-- а money/quantity могли быть TEXT. Сейчас 00001 создаёт NUMERIC, 00002 — таблицу
-- без FK. ALTER TYPE здесь нельзя: Postgres берёт ACCESS EXCLUSIVE на orders
-- даже если тип уже NUMERIC (простой на время миграции).
-- DROP CONSTRAINT IF EXISTS безопасен и нужен только старым БД.

ALTER TABLE idempotency_keys
    DROP CONSTRAINT IF EXISTS idempotency_keys_order_id_fkey;

-- +goose Down
-- Не возвращаем TEXT и не вешаем FK: это ломало бы текущий 00001/00002
-- и reserve-before-create.
SELECT 1;
