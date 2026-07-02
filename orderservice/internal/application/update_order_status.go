package application

import (
	"context"
	"fmt"
	"strings"

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
	orders domain.OrderRepository
	hub    *UpdateHub
}

// NewUpdateOrderStatus создаёт use case UpdateOrderStatus.
func NewUpdateOrderStatus(orders domain.OrderRepository, hub *UpdateHub) *UpdateOrderStatus {
	return &UpdateOrderStatus{orders: orders, hub: hub}
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

	order.Status = input.Status
	if err := uc.orders.Save(ctx, order); err != nil {
		return err
	}

	if uc.hub != nil {
		uc.hub.Publish(order.ID, order.Status)
	}

	return nil
}
