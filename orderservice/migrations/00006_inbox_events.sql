-- +goose Up
-- Inbox входящих команд matching-engine (G6): уникальный event_id гасит дубли Kafka.
-- INSERT в inbox, смена статуса и outbox-событие идут в одной транзакции.
CREATE TABLE IF NOT EXISTS inbox_events (
    event_id UUID PRIMARY KEY,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS inbox_events;
