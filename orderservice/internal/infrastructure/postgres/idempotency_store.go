package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/jackc/pgx/v5"
)

// IdempotencyStore хранит idempotency keys в PostgreSQL.
type IdempotencyStore struct {
	db *DB
}

// NewIdempotencyStore создаёт postgres idempotency store.
func NewIdempotencyStore(db *DB) *IdempotencyStore {
	return &IdempotencyStore{db: db}
}

// GetOrderID возвращает order_id по idempotency key.
func (s *IdempotencyStore) GetOrderID(ctx context.Context, userID, key string) (string, bool, error) {
	var orderID string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT order_id
		FROM idempotency_keys
		WHERE user_id = $1 AND idempotency_key = $2
	`, userID, key).Scan(&orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get idempotency key: %w", err)
	}
	return orderID, true, nil
}

// Save сохраняет idempotency key.
func (s *IdempotencyStore) Save(ctx context.Context, userID, key, orderID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO idempotency_keys (user_id, idempotency_key, order_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, idempotency_key) DO NOTHING
	`, userID, key, orderID)
	if err != nil {
		return fmt.Errorf("save idempotency key: %w", err)
	}
	return nil
}

var _ domain.IdempotencyStore = (*IdempotencyStore)(nil)
