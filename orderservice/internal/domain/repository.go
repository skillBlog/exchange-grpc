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
	// ListByUserID возвращает страницу ордеров пользователя по курсору id.
	// limit — максимальное число записей; afterID — exclusive cursor (пустой = с начала).
	ListByUserID(ctx context.Context, userID string, limit int, afterID string) ([]Order, error)
	UpdateStatus(ctx context.Context, id string, status OrderStatus, updatedAt time.Time) error
}

// IdempotencyStore хранит соответствие idempotency_key → order_id.
type IdempotencyStore interface {
	GetOrderID(ctx context.Context, userID, key string) (string, bool, error)
	// Reserve пытается зарезервировать ключ под orderID до создания ордера.
	// reserved=true — ключ занят нами; reserved=false и existingOrderID!="" — ключ уже был.
	Reserve(ctx context.Context, userID, key, orderID string) (reserved bool, existingOrderID string, err error)
	// Release снимает резерв ключа, если создание ордера после Reserve не удалось.
	Release(ctx context.Context, userID, key string) error
}
