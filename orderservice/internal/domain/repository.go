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

// IdempotencyKeyStatus — жизненный цикл idempotency-ключа.
type IdempotencyKeyStatus string

const (
	IdempotencyStatusReserved  IdempotencyKeyStatus = "reserved"
	IdempotencyStatusCompleted IdempotencyKeyStatus = "completed"
	IdempotencyStatusFailed    IdempotencyKeyStatus = "failed"
)

// IdempotencyStore хранит соответствие idempotency_key → order_id со статусной моделью.
type IdempotencyStore interface {
	// GetOrderID возвращает order_id только для completed и неистёкших ключей.
	GetOrderID(ctx context.Context, userID, key string) (string, bool, error)
	// Reserve резервирует ключ под orderID (или перезаписывает failed/expired).
	// reserved=true — ключ наш; reserved=false и existingOrderID!="" — ключ уже занят.
	Reserve(ctx context.Context, userID, key, orderID string) (reserved bool, existingOrderID string, err error)
	// Complete помечает ключ завершённым после успешного Create.
	Complete(ctx context.Context, userID, key string) error
	// Fail помечает ключ failed после ошибки Create (без DELETE — безопасный retry).
	Fail(ctx context.Context, userID, key string) error
}
