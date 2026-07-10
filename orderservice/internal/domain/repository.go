package domain

import (
	"context"
	"time"
)

// OrderRepository предоставляет доступ к хранилищу ордеров.
type OrderRepository interface {
	Create(ctx context.Context, order Order) error
	GetByID(ctx context.Context, id string) (Order, error)
	GetByIDAndUserID(ctx context.Context, id, userID string) (Order, error)
	ListByUserID(ctx context.Context, userID string) ([]Order, error)
	UpdateStatus(ctx context.Context, id string, status OrderStatus, updatedAt time.Time) error
}

// IdempotencyStore хранит соответствие idempotency_key → order_id.
type IdempotencyStore interface {
	GetOrderID(ctx context.Context, userID, key string) (string, bool, error)
	Save(ctx context.Context, userID, key, orderID string) error
}
