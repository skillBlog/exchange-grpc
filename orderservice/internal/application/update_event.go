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
