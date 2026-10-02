-- +goose Up
-- Транзакционный outbox: ордер и событие пишутся в одной транзакции (G2).
-- Relay забирает processed_at IS NULL (G3). Kafka producer — после relay (G4).
CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY,
    aggregate_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    attempts INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS outbox_events_unprocessed_created_at_idx
    ON outbox_events (created_at)
    WHERE processed_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS outbox_events_unprocessed_created_at_idx;
DROP TABLE IF EXISTS outbox_events;
