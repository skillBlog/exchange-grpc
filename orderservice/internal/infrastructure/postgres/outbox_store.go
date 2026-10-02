package postgres

import (
	"context"
	"fmt"

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

// ClaimBatch блокирует пачку unprocessed строк до конца текущей транзакции.
func (s *OutboxStore) ClaimBatch(ctx context.Context, limit int) ([]application.OutboxEvent, error) {
	if _, ok := TxFromContext(ctx); !ok {
		return nil, fmt.Errorf("claim outbox: requires transaction")
	}
	if limit <= 0 {
		return nil, nil
	}

	rows, err := querierFrom(ctx, s.db.Pool).Query(ctx, `
		SELECT id, aggregate_id, event_type, payload, created_at
		FROM outbox_events
		WHERE processed_at IS NULL
		ORDER BY created_at ASC, id ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
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

// MarkProcessed ставит processed_at. Вызывать после Publish — at-least-once.
func (s *OutboxStore) MarkProcessed(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	q := querierFrom(ctx, s.db.Pool)
	for _, id := range ids {
		if _, err := q.Exec(ctx, `
			UPDATE outbox_events
			SET processed_at = NOW()
			WHERE id = $1 AND processed_at IS NULL
		`, id); err != nil {
			return fmt.Errorf("mark outbox processed: %w", err)
		}
	}
	return nil
}

// RecordAttempt увеличивает attempts у события, которое не удалось разобрать/отправить.
func (s *OutboxStore) RecordAttempt(ctx context.Context, id string) error {
	_, err := querierFrom(ctx, s.db.Pool).Exec(ctx, `
		UPDATE outbox_events
		SET attempts = attempts + 1
		WHERE id = $1
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
