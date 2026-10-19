package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
)

// OutboxStore пишет события в outbox_events в текущей транзакции из ctx.
type OutboxStore struct {
	db *DB
}

// NewOutboxStore создаёт postgres outbox store.
func NewOutboxStore(db *DB) *OutboxStore {
	return &OutboxStore{db: db}
}

// Append вставляет строку outbox. processed_at остаётся NULL до relay (G3).
func (s *OutboxStore) Append(ctx context.Context, event application.OutboxEvent) error {
	_, err := querierFrom(ctx, s.db.Pool).Exec(ctx, `
		INSERT INTO outbox_events (id, aggregate_id, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, event.ID, event.AggregateID, event.EventType, event.Payload, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}

// ClaimBatch ставит claimed_at на пачку свободных строк и возвращает их.
// Вызывать внутри WithinTx: FOR UPDATE SKIP LOCKED держится только до commit этой транзакции.
// После commit строку не возьмёт другая реплика, пока claim не снят или не протух.
func (s *OutboxStore) ClaimBatch(ctx context.Context, limit int) ([]application.OutboxEvent, error) {
	if _, ok := TxFromContext(ctx); !ok {
		return nil, fmt.Errorf("claim outbox: requires transaction")
	}
	if limit <= 0 {
		return nil, nil
	}

	rows, err := querierFrom(ctx, s.db.Pool).Query(ctx, `
		WITH picked AS (
			SELECT id
			FROM outbox_events
			WHERE processed_at IS NULL
			  AND claimed_at IS NULL
			ORDER BY created_at ASC, id ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		),
		updated AS (
			UPDATE outbox_events AS e
			SET claimed_at = NOW()
			FROM picked
			WHERE e.id = picked.id
			RETURNING e.id, e.aggregate_id, e.event_type, e.payload, e.created_at
		)
		SELECT id, aggregate_id, event_type, payload, created_at
		FROM updated
		ORDER BY created_at ASC, id ASC
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("claim outbox events: %w", err)
	}
	defer rows.Close()

	events := make([]application.OutboxEvent, 0, limit)
	for rows.Next() {
		var event application.OutboxEvent
		if err := rows.Scan(&event.ID, &event.AggregateID, &event.EventType, &event.Payload, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("claim outbox events: %w", err)
	}
	return events, nil
}

// ReleaseStaleClaims снимает claim старше olderThan. Строка снова попадает в очередь.
func (s *OutboxStore) ReleaseStaleClaims(ctx context.Context, olderThan time.Time) (int, error) {
	tag, err := querierFrom(ctx, s.db.Pool).Exec(ctx, `
		UPDATE outbox_events
		SET claimed_at = NULL
		WHERE processed_at IS NULL
		  AND claimed_at IS NOT NULL
		  AND claimed_at < $1
	`, olderThan)
	if err != nil {
		return 0, fmt.Errorf("release stale outbox claims: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// MarkProcessed ставит processed_at одним UPDATE. Вызывать после Publish — at-least-once.
func (s *OutboxStore) MarkProcessed(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := querierFrom(ctx, s.db.Pool).Exec(ctx, `
		UPDATE outbox_events
		SET processed_at = NOW(),
		    claimed_at = NULL
		WHERE id IN (SELECT unnest($1::text[])::uuid)
		  AND processed_at IS NULL
	`, ids)
	if err != nil {
		return fmt.Errorf("mark outbox processed: %w", err)
	}
	return nil
}

// RecordAttempt увеличивает attempts и снимает claim, чтобы событие взяли на следующем тике.
func (s *OutboxStore) RecordAttempt(ctx context.Context, id string) error {
	_, err := querierFrom(ctx, s.db.Pool).Exec(ctx, `
		UPDATE outbox_events
		SET attempts = attempts + 1,
		    claimed_at = NULL
		WHERE id = $1
		  AND processed_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("record outbox attempt: %w", err)
	}
	return nil
}

var (
	_ application.OutboxStore      = (*OutboxStore)(nil)
	_ application.OutboxRelayStore = (*OutboxStore)(nil)
)
