package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/memory"
	"github.com/google/uuid"
)

type recordingNotifier struct {
	events []application.UpdateEvent
}

func (n *recordingNotifier) Publish(event application.UpdateEvent) {
	n.events = append(n.events, event)
}

func TestOutboxRelay_publishesAndMarksProcessed(t *testing.T) {
	outbox := memory.NewOutboxStore()
	now := time.Now().UTC()
	orderID := domain.NewOrderID()
	event, err := marshalTestOutboxEvent(orderID, "11111111-1111-1111-1111-111111111111", "BTC-USDT", domain.OrderStatusCreated, now)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	if err := outbox.Append(context.Background(), event); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	notifier := &recordingNotifier{}
	relay := application.NewOutboxRelay(memory.NewTxManager(), outbox, notifier, nil, nil, time.Millisecond, 10)

	processed, claimed, err := relay.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if claimed != 1 || processed != 1 {
		t.Fatalf("processed=%d claimed=%d, want 1 1", processed, claimed)
	}
	if len(notifier.events) != 1 {
		t.Fatalf("published %d events, want 1", len(notifier.events))
	}
	if notifier.events[0].OrderID != orderID {
		t.Fatalf("order_id = %q, want %q", notifier.events[0].OrderID, orderID)
	}
	if notifier.events[0].Status != domain.OrderStatusCreated {
		t.Fatalf("status = %q", notifier.events[0].Status)
	}

	processed, claimed, err = relay.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("second RunOnce() error = %v", err)
	}
	if claimed != 0 || processed != 0 {
		t.Fatalf("second poll processed=%d claimed=%d, want 0 0", processed, claimed)
	}
	if len(notifier.events) != 1 {
		t.Fatalf("republished events: %d", len(notifier.events))
	}
}

func TestOutboxRelay_invalidPayloadRecordsAttempt(t *testing.T) {
	outbox := memory.NewOutboxStore()
	bad := application.OutboxEvent{
		ID:          uuid.Must(uuid.NewV7()).String(),
		AggregateID: domain.NewOrderID(),
		EventType:   application.EventTypeOrderCreated,
		Payload:     json.RawMessage(`{`),
		CreatedAt:   time.Now().UTC(),
	}
	if err := outbox.Append(context.Background(), bad); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	notifier := &recordingNotifier{}
	relay := application.NewOutboxRelay(memory.NewTxManager(), outbox, notifier, nil, nil, time.Millisecond, 10)

	processed, claimed, err := relay.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if claimed != 1 || processed != 0 {
		t.Fatalf("processed=%d claimed=%d, want 0 1", processed, claimed)
	}
	if len(notifier.events) != 0 {
		t.Fatalf("published %d events, want 0", len(notifier.events))
	}

	if err := relay.Drain(context.Background()); err != nil {
		t.Fatalf("Drain() error = %v", err)
	}
	if len(notifier.events) != 0 {
		t.Fatal("Drain must not busy-loop poison payload into publish")
	}
}

func TestOutboxRelay_missingOrderIDIsInvalid(t *testing.T) {
	outbox := memory.NewOutboxStore()
	event := application.OutboxEvent{
		ID:          uuid.Must(uuid.NewV7()).String(),
		AggregateID: domain.NewOrderID(),
		EventType:   application.EventTypeOrderCreated,
		Payload:     json.RawMessage(`{"user_id":"11111111-1111-1111-1111-111111111111"}`),
		CreatedAt:   time.Now().UTC(),
	}
	if err := outbox.Append(context.Background(), event); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	notifier := &recordingNotifier{}
	relay := application.NewOutboxRelay(memory.NewTxManager(), outbox, notifier, nil, nil, time.Millisecond, 10)
	processed, claimed, err := relay.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if processed != 0 || claimed != 1 {
		t.Fatalf("processed=%d claimed=%d, want 0 1", processed, claimed)
	}
	if len(notifier.events) != 0 {
		t.Fatalf("published %d, want 0", len(notifier.events))
	}
}

func TestParseOutboxUpdateEvent_roundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	src := application.UpdateEvent{
		OrderID:   domain.NewOrderID(),
		UserID:    "11111111-1111-1111-1111-111111111111",
		MarketID:  "ETH-USDT",
		Status:    domain.OrderStatusFilled,
		UpdatedAt: now,
	}
	raw, err := json.Marshal(map[string]any{
		"order_id":   src.OrderID,
		"user_id":    src.UserID,
		"market_id":  src.MarketID,
		"status":     string(src.Status),
		"updated_at": src.UpdatedAt,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := application.ParseOutboxUpdateEvent(application.OutboxEvent{Payload: raw})
	if err != nil {
		t.Fatalf("ParseOutboxUpdateEvent() error = %v", err)
	}
	if got.OrderID != src.OrderID || got.UserID != src.UserID || got.MarketID != src.MarketID || got.Status != src.Status {
		t.Fatalf("got %+v, want %+v", got, src)
	}
	if !got.UpdatedAt.Equal(src.UpdatedAt) {
		t.Fatalf("updated_at = %v, want %v", got.UpdatedAt, src.UpdatedAt)
	}
}

func TestParseOutboxUpdateEvent_invalidJSON(t *testing.T) {
	_, err := application.ParseOutboxUpdateEvent(application.OutboxEvent{Payload: json.RawMessage(`{`)})
	if err == nil {
		t.Fatal("expected error")
	}
}

type recordingPublisher struct {
	events []application.OutboxEvent
}

func (p *recordingPublisher) Publish(_ context.Context, event application.OutboxEvent) error {
	p.events = append(p.events, event)
	return nil
}

type failPublisher struct {
	err   error
	calls int
}

func (p *failPublisher) Publish(context.Context, application.OutboxEvent) error {
	p.calls++
	return p.err
}

func TestOutboxRelay_publisherMarksWithoutHub(t *testing.T) {
	outbox := memory.NewOutboxStore()
	now := time.Now().UTC()
	orderID := domain.NewOrderID()
	event, err := marshalTestOutboxEvent(orderID, "11111111-1111-1111-1111-111111111111", "BTC-USDT", domain.OrderStatusCreated, now)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	if err := outbox.Append(context.Background(), event); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	hub := &recordingNotifier{}
	bus := &recordingPublisher{}
	relay := application.NewOutboxRelay(memory.NewTxManager(), outbox, hub, bus, nil, time.Millisecond, 10)

	processed, claimed, err := relay.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if claimed != 1 || processed != 1 {
		t.Fatalf("processed=%d claimed=%d, want 1 1", processed, claimed)
	}
	if len(bus.events) != 1 || bus.events[0].ID != event.ID {
		t.Fatalf("publisher events = %+v", bus.events)
	}
	if len(hub.events) != 0 {
		t.Fatalf("hub events = %d, want 0 (G5: hub feeds from kafka consumer)", len(hub.events))
	}

	processed, claimed, err = relay.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("second RunOnce() error = %v", err)
	}
	if claimed != 0 || processed != 0 {
		t.Fatalf("second poll processed=%d claimed=%d, want 0 0", processed, claimed)
	}
}

func TestOutboxRelay_publisherFailureSkipsHubAndMark(t *testing.T) {
	outbox := memory.NewOutboxStore()
	now := time.Now().UTC()
	orderID := domain.NewOrderID()
	event, err := marshalTestOutboxEvent(orderID, "11111111-1111-1111-1111-111111111111", "BTC-USDT", domain.OrderStatusCreated, now)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	if err := outbox.Append(context.Background(), event); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	hub := &recordingNotifier{}
	bus := &failPublisher{err: errors.New("broker down")}
	relay := application.NewOutboxRelay(memory.NewTxManager(), outbox, hub, bus, nil, time.Millisecond, 10)

	processed, claimed, err := relay.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if claimed != 1 || processed != 0 {
		t.Fatalf("processed=%d claimed=%d, want 0 1", processed, claimed)
	}
	if bus.calls != 1 {
		t.Fatalf("publisher calls = %d, want 1", bus.calls)
	}
	if len(hub.events) != 0 {
		t.Fatalf("hub published %d events, want 0", len(hub.events))
	}

	processed, claimed, err = relay.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("retry RunOnce() error = %v", err)
	}
	if claimed != 1 || processed != 0 {
		t.Fatalf("retry processed=%d claimed=%d, want 0 1", processed, claimed)
	}
	if bus.calls != 2 {
		t.Fatalf("publisher calls = %d, want 2", bus.calls)
	}
}

func marshalTestOutboxEvent(orderID, userID, marketID string, status domain.OrderStatus, now time.Time) (application.OutboxEvent, error) {
	payload, err := json.Marshal(map[string]any{
		"order_id":   orderID,
		"user_id":    userID,
		"market_id":  marketID,
		"status":     string(status),
		"updated_at": now,
	})
	if err != nil {
		return application.OutboxEvent{}, err
	}
	return application.OutboxEvent{
		ID:          uuid.Must(uuid.NewV7()).String(),
		AggregateID: orderID,
		EventType:   application.EventTypeOrderCreated,
		Payload:     payload,
		CreatedAt:   now,
	}, nil
}
