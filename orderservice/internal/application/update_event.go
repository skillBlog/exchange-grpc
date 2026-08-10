package application

import (
	"time"

	"github.com/exchange-grpc/orderservice/internal/domain"
)

// UpdateEvent отправляется подписчикам на обновления ордера.
type UpdateEvent struct {
	OrderID   string
	Status    domain.OrderStatus
	UpdatedAt time.Time
}
