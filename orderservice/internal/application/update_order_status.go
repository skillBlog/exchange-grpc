package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/exchange-grpc/orderservice/internal/domain"
)

// UpdateOrderStatusInput — запрос на смену статуса ордера.
type UpdateOrderStatusInput struct {
	OrderID string
	UserID  string
	Status  domain.OrderStatus
}

// UpdateOrderStatus меняет статус ордера и публикует событие обновления.
type UpdateOrderStatus struct {
	orders   domain.OrderRepository
	notifier OrderNotifier
	now      func() time.Time
}

// NewUpdateOrderStatus создаёт use case UpdateOrderStatus.
func NewUpdateOrderStatus(orders domain.OrderRepository, notifier OrderNotifier) *UpdateOrderStatus {
	return &UpdateOrderStatus{
		orders:   orders,
		notifier: notifier,
		now:      time.Now,
	}
}

// Execute обновляет статус ордера, если он принадлежит пользователю.
func (uc *UpdateOrderStatus) Execute(ctx context.Context, input UpdateOrderStatusInput) error {
	if strings.TrimSpace(input.OrderID) == "" {
		return fmt.Errorf("%w: order_id is required", domain.ErrInvalidArgument)
	}
	if strings.TrimSpace(input.UserID) == "" {
		return fmt.Errorf("%w: user_id is required", domain.ErrInvalidArgument)
	}
	if strings.TrimSpace(string(input.Status)) == "" {
		return fmt.Errorf("%w: status is required", domain.ErrInvalidArgument)
	}

	order, err := uc.orders.GetByIDAndUserID(ctx, input.OrderID, input.UserID)
	if err != nil {
		return err
	}

	if err := domain.ValidateTransition(order.Status, input.Status); err != nil {
		return err
	}

	now := uc.now().UTC()
	if err := uc.orders.UpdateStatus(ctx, order.ID, input.Status, now); err != nil {
		return err
	}

	if uc.notifier != nil {
		uc.notifier.Publish(order.ID, input.Status)
	}
	return nil
}
