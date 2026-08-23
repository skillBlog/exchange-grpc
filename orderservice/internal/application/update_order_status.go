package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/shared/tracing"
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
func (uc *UpdateOrderStatus) Execute(ctx context.Context, input UpdateOrderStatusInput) (err error) {
	ctx, span := tracing.Start(ctx, "order.UpdateOrderStatus",
		tracing.Attr("order.id", strings.TrimSpace(input.OrderID)),
	)
	defer tracing.End(span, &err)

	orderID, err := domain.ParseUUID(input.OrderID, "order_id")
	if err != nil {
		return err
	}
	userID, err := domain.ParseUUID(input.UserID, "user_id")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(input.Status)) == "" {
		return fmt.Errorf("%w: status is required", domain.ErrInvalidArgument)
	}

	order, err := uc.orders.GetByIDAndUserID(ctx, orderID, userID)
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
		uc.notifier.Publish(order.ID, input.Status, now)
	}
	return nil
}
