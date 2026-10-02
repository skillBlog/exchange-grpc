package postgres

import (
	"context"
	"fmt"

	"github.com/exchange-grpc/orderservice/internal/application"
)

// InboxStore пишет event_id в inbox_events в текущей транзакции из ctx.
type InboxStore struct {
	db *DB
}

// NewInboxStore создаёт postgres inbox store.
func NewInboxStore(db *DB) *InboxStore {
	return &InboxStore{db: db}
}

// InsertIfNew вставляет event_id. Дубль (PK) — inserted=false, без ошибки.
func (s *InboxStore) InsertIfNew(ctx context.Context, eventID string) (bool, error) {
	tag, err := querierFrom(ctx, s.db.Pool).Exec(ctx, `
		INSERT INTO inbox_events (event_id)
		VALUES ($1)
		ON CONFLICT (event_id) DO NOTHING
	`, eventID)
	if err != nil {
		return false, fmt.Errorf("insert inbox event: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

var _ application.InboxStore = (*InboxStore)(nil)
