package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/shared/tracing"
)

// StreamUserOrderUpdatesInput идентифицирует подписку на поток ордеров пользователя.
type StreamUserOrderUpdatesInput struct {
	UserID   string
	MarketID string
	Status   domain.OrderStatus
}

// StreamUserOrderUpdates стримит обновления всех ордеров пользователя до отмены контекста.
type StreamUserOrderUpdates struct {
	hub OrderUpdateHub
}

// NewStreamUserOrderUpdates создаёт use case StreamUserOrderUpdates.
func NewStreamUserOrderUpdates(hub OrderUpdateHub) *StreamUserOrderUpdates {
	return &StreamUserOrderUpdates{hub: hub}
}

// Execute стримит live-обновления ордеров пользователя. Стрим не закрывается на terminal-статусе.
func (uc *StreamUserOrderUpdates) Execute(ctx context.Context, input StreamUserOrderUpdatesInput, send func(UpdateEvent) error) (err error) {
	ctx, span := tracing.Start(ctx, "order.StreamUserOrderUpdates",
		tracing.Attr("user_id", strings.TrimSpace(input.UserID)),
	)
	defer tracing.End(span, &err)

	userID, err := domain.ParseUUID(input.UserID, "user_id")
	if err != nil {
		return err
	}
	marketID := strings.TrimSpace(input.MarketID)

	if uc.hub == nil {
		return fmt.Errorf("order update hub is not configured")
	}

	updates, unsubscribe := uc.hub.SubscribeUser(userID)
	defer unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			if !matchesUserStreamFilter(update, marketID, input.Status) {
				continue
			}
			if err = send(update); err != nil {
				return err
			}
		}
	}
}

func matchesUserStreamFilter(event UpdateEvent, marketID string, status domain.OrderStatus) bool {
	if marketID != "" && event.MarketID != marketID {
		return false
	}
	if status != "" && event.Status != status {
		return false
	}
	return true
}
