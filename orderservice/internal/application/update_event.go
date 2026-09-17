package application

import (
	"time"

	"github.com/exchange-grpc/orderservice/internal/domain"
)

// UpdateEvent отправляется подписчикам на обновления ордера.
type UpdateEvent struct {
	OrderID   string
	UserID    string
	MarketID  string
	Status    domain.OrderStatus
	UpdatedAt time.Time
}

// UserStreamFilter сужает подписку на ордера пользователя.
// Пустые поля не ограничивают выборку.
type UserStreamFilter struct {
	MarketID string
	Status   domain.OrderStatus
}

// Matches сообщает, проходит ли событие фильтр подписки.
func (f UserStreamFilter) Matches(event UpdateEvent) bool {
	if f.MarketID != "" && event.MarketID != f.MarketID {
		return false
	}
	if f.Status != "" && event.Status != f.Status {
		return false
	}
	return true
}
