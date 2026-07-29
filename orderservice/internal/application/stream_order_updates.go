package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/shared/tracing"
)

// StreamOrderUpdatesInput идентифицирует подписку на поток обновлений ордера.
type StreamOrderUpdatesInput struct {
	OrderID string
	UserID  string
}

// StreamOrderUpdates передаёт текущий и последующие статусы ордера.
type StreamOrderUpdates struct {
	orders domain.OrderRepository
	hub    OrderUpdateHub
}

// NewStreamOrderUpdates создаёт use case StreamOrderUpdates.
func NewStreamOrderUpdates(orders domain.OrderRepository, hub OrderUpdateHub) *StreamOrderUpdates {
	return &StreamOrderUpdates{orders: orders, hub: hub}
}

// Execute отправляет текущий статус, затем стримит обновления из hub до отмены контекста.
func (uc *StreamOrderUpdates) Execute(ctx context.Context, input StreamOrderUpdatesInput, send func(UpdateEvent) error) (err error) {
	ctx, span := tracing.Start(ctx, "order.StreamOrderUpdates",
		tracing.Attr("order.id", strings.TrimSpace(input.OrderID)),
	)
	defer tracing.End(span, &err)

	orderID := strings.TrimSpace(input.OrderID)
	userID := strings.TrimSpace(input.UserID)
	if orderID == "" {
		return fmt.Errorf("%w: order_id is required", domain.ErrInvalidArgument)
	}
	if userID == "" {
		return fmt.Errorf("%w: user_id is required", domain.ErrInvalidArgument)
	}

	order, err := uc.orders.GetByIDAndUserID(ctx, orderID, userID)
	if err != nil {
		return err
	}

	if err = send(UpdateEvent{OrderID: order.ID, Status: order.Status}); err != nil {
		return err
	}

	if uc.hub == nil {
		<-ctx.Done()
		return ctx.Err()
	}

	updates, unsubscribe := uc.hub.Subscribe(order.ID)
	defer unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			if err = send(update); err != nil {
				return err
			}
		}
	}
}
