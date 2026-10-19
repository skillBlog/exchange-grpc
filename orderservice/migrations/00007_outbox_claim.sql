-- +goose Up
-- Claim коммитится в claimed_at до Kafka Produce.
-- Индекс совпадает с выборкой claim: ещё не обработано, ещё не взято, порядок created_at, id.
ALTER TABLE outbox_events
    ADD COLUMN IF NOT EXISTS claimed_at TIMESTAMPTZ;

DROP INDEX IF EXISTS outbox_events_unprocessed_created_at_idx;

CREATE INDEX IF NOT EXISTS outbox_events_claim_idx
    ON outbox_events (created_at, id)
    WHERE processed_at IS NULL AND claimed_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS outbox_events_claim_idx;

CREATE INDEX IF NOT EXISTS outbox_events_unprocessed_created_at_idx
    ON outbox_events (created_at)
    WHERE processed_at IS NULL;

ALTER TABLE outbox_events
    DROP COLUMN IF EXISTS claimed_at;
