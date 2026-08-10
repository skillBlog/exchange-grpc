package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/jackc/pgx/v5"
)

const defaultIdempotencyTTL = 24 * time.Hour

// IdempotencyStore хранит idempotency keys в PostgreSQL.
type IdempotencyStore struct {
	db  *DB
	ttl time.Duration
	now func() time.Time
}

// NewIdempotencyStore создаёт postgres idempotency store.
func NewIdempotencyStore(db *DB, ttl time.Duration) *IdempotencyStore {
	if ttl <= 0 {
		ttl = defaultIdempotencyTTL
	}
	return &IdempotencyStore{db: db, ttl: ttl, now: time.Now}
}

// GetOrderID возвращает order_id для completed и ещё живого ключа.
func (s *IdempotencyStore) GetOrderID(ctx context.Context, userID, key string) (string, bool, error) {
	var orderID string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT order_id
		FROM idempotency_keys
		WHERE user_id = $1
			AND idempotency_key = $2
			AND status = $3
			AND expires_at > NOW()
	`, userID, key, string(domain.IdempotencyStatusCompleted)).Scan(&orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get idempotency key: %w", err)
	}
	return orderID, true, nil
}

// Reserve резервирует ключ одним запросом с RETURNING.
// Перезаписывает failed или истёкшие ключи; иначе возвращает существующий order_id.
func (s *IdempotencyStore) Reserve(ctx context.Context, userID, key, orderID string) (bool, string, error) {
	expiresAt := s.now().UTC().Add(s.ttl)

	var reservedOrderID string
	err := s.db.Pool.QueryRow(ctx, `
		INSERT INTO idempotency_keys (user_id, idempotency_key, order_id, status, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (user_id, idempotency_key) DO UPDATE
		SET order_id = EXCLUDED.order_id,
			status = EXCLUDED.status,
			expires_at = EXCLUDED.expires_at,
			created_at = NOW()
		WHERE idempotency_keys.status = $6
			OR idempotency_keys.expires_at <= NOW()
		RETURNING order_id
	`, userID, key, orderID, string(domain.IdempotencyStatusReserved), expiresAt, string(domain.IdempotencyStatusFailed)).Scan(&reservedOrderID)
	if err == nil {
		return true, "", nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, "", fmt.Errorf("reserve idempotency key: %w", err)
	}

	var existingOrderID string
	var status string
	err = s.db.Pool.QueryRow(ctx, `
		SELECT order_id, status
		FROM idempotency_keys
		WHERE user_id = $1 AND idempotency_key = $2
	`, userID, key).Scan(&existingOrderID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, "", fmt.Errorf("reserve idempotency key: conflict without existing row")
	}
	if err != nil {
		return false, "", fmt.Errorf("reserve idempotency key lookup: %w", err)
	}
	return false, existingOrderID, nil
}

// Complete помечает ключ как успешно использованный.
func (s *IdempotencyStore) Complete(ctx context.Context, userID, key string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE idempotency_keys
		SET status = $3
		WHERE user_id = $1 AND idempotency_key = $2 AND status = $4
	`, userID, key, string(domain.IdempotencyStatusCompleted), string(domain.IdempotencyStatusReserved))
	if err != nil {
		return fmt.Errorf("complete idempotency key: %w", err)
	}
	return nil
}

// Fail помечает ключ failed после ошибки Create (без DELETE).
func (s *IdempotencyStore) Fail(ctx context.Context, userID, key string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE idempotency_keys
		SET status = $3
		WHERE user_id = $1 AND idempotency_key = $2 AND status = $4
	`, userID, key, string(domain.IdempotencyStatusFailed), string(domain.IdempotencyStatusReserved))
	if err != nil {
		return fmt.Errorf("fail idempotency key: %w", err)
	}
	return nil
}

var _ domain.IdempotencyStore = (*IdempotencyStore)(nil)
