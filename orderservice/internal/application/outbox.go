package application

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/google/uuid"
)

const (
	EventTypeOrderCreated       = "order.created"
	EventTypeOrderStatusChanged = "order.status_changed"
)

// OutboxEvent — строка transactional outbox. Relay (G3) забирает unprocessed.
type OutboxEvent struct {
	ID          string
	AggregateID string
	EventType   string
	Payload     json.RawMessage
	CreatedAt   time.Time
}

type orderOutboxPayload struct {
	OrderID   string    `json:"order_id"`
	UserID    string    `json:"user_id"`
	MarketID  string    `json:"market_id"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ParseOutboxUpdateEvent достаёт UpdateEvent из payload, который пишет Create/Update.
func ParseOutboxUpdateEvent(event OutboxEvent) (UpdateEvent, error) {
	var payload orderOutboxPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return UpdateEvent{}, fmt.Errorf("decode outbox payload: %w", err)
	}
	if payload.OrderID == "" {
		return UpdateEvent{}, fmt.Errorf("%w: outbox order_id is required", domain.ErrInvalidArgument)
	}
	return UpdateEvent{
		OrderID:   payload.OrderID,
		UserID:    payload.UserID,
		MarketID:  payload.MarketID,
		Status:    domain.OrderStatus(payload.Status),
		UpdatedAt: payload.UpdatedAt,
	}, nil
}

func newOrderOutboxEvent(eventType string, event UpdateEvent) (OutboxEvent, error) {
	payload, err := json.Marshal(orderOutboxPayload{
		OrderID:   event.OrderID,
		UserID:    event.UserID,
		MarketID:  event.MarketID,
		Status:    string(event.Status),
		UpdatedAt: event.UpdatedAt,
	})
	if err != nil {
		return OutboxEvent{}, fmt.Errorf("marshal outbox payload: %w", err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return OutboxEvent{}, fmt.Errorf("outbox event id: %w", err)
	}
	return OutboxEvent{
		ID:          id.String(),
		AggregateID: event.OrderID,
		EventType:   eventType,
		Payload:     payload,
		CreatedAt:   event.UpdatedAt,
	}, nil
}

type nopTxManager struct{}

func (nopTxManager) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type nopOutbox struct{}

func (nopOutbox) Append(context.Context, OutboxEvent) error {
	return nil
}

var (
	_ TxManager   = nopTxManager{}
	_ OutboxStore = nopOutbox{}
)
