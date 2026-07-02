package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/exchange-grpc/orderservice/internal/domain"
)

// StreamOrderUpdatesInput идентифицирует подписку на поток обновлений ордера.
type StreamOrderUpdatesInput struct {
	OrderID string
	UserID  string
}

// StreamOrderUpdates передаёт текущий и последующие статусы ордера.
type StreamOrderUpdates struct {
	orders domain.OrderRepository
	hub    *UpdateHub
}

// NewStreamOrderUpdates создаёт use case StreamOrderUpdates.
func NewStreamOrderUpdates(orders domain.OrderRepository, hub *UpdateHub) *StreamOrderUpdates {
	return &StreamOrderUpdates{orders: orders, hub: hub}
}

// Execute отправляет текущий статус, затем стримит обновления из hub до отмены контекста.
func (uc *StreamOrderUpdates) Execute(ctx context.Context, input StreamOrderUpdatesInput, send func(UpdateEvent) error) error {
	if strings.TrimSpace(input.OrderID) == "" {
		return fmt.Errorf("%w: order_id is required", domain.ErrInvalidArgument)
	}
	if strings.TrimSpace(input.UserID) == "" {
		return fmt.Errorf("%w: user_id is required", domain.ErrInvalidArgument)
	}

	order, err := uc.orders.GetByIDAndUserID(ctx, input.OrderID, input.UserID)
	if err != nil {
		return err
	}

	if err := send(UpdateEvent{OrderID: order.ID, Status: order.Status}); err != nil {
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
			if err := send(update); err != nil {
				return err
			}
		}
	}
}
