package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	kafkago "github.com/segmentio/kafka-go"
)

type recordingHub struct {
	events []application.UpdateEvent
}

func (h *recordingHub) Publish(event application.UpdateEvent) {
	h.events = append(h.events, event)
}

func TestDispatchToHub_roundTripFromProducerMessage(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	orderID := "11111111-1111-4111-8111-111111111111"
	payload, err := json.Marshal(map[string]any{
		"order_id":   orderID,
		"user_id":    "22222222-2222-4222-8222-222222222222",
		"market_id":  "BTC-USDT",
		"status":     "filled",
		"updated_at": now,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	msg, err := messageFromEvent(context.Background(), application.OutboxEvent{
		ID:          "evt-1",
		AggregateID: orderID,
		EventType:   application.EventTypeOrderStatusChanged,
		Payload:     payload,
		CreatedAt:   now,
	})
	if err != nil {
		t.Fatalf("messageFromEvent() error = %v", err)
	}

	hub := &recordingHub{}
	if err := DispatchToHub(hub, msg.Value); err != nil {
		t.Fatalf("DispatchToHub() error = %v", err)
	}
	if len(hub.events) != 1 {
		t.Fatalf("hub events = %d, want 1", len(hub.events))
	}
	got := hub.events[0]
	if got.OrderID != orderID {
		t.Fatalf("order_id = %q", got.OrderID)
	}
	if got.UserID != "22222222-2222-4222-8222-222222222222" {
		t.Fatalf("user_id = %q", got.UserID)
	}
	if got.MarketID != "BTC-USDT" {
		t.Fatalf("market_id = %q", got.MarketID)
	}
	if got.Status != domain.OrderStatusFilled {
		t.Fatalf("status = %q", got.Status)
	}
}

func TestDispatchToHub_invalidJSON(t *testing.T) {
	hub := &recordingHub{}
	if err := DispatchToHub(hub, []byte(`{`)); err == nil {
		t.Fatal("expected error")
	}
	if len(hub.events) != 0 {
		t.Fatalf("published %d, want 0", len(hub.events))
	}
}

func TestDispatchToHub_missingOrderID(t *testing.T) {
	hub := &recordingHub{}
	body, err := json.Marshal(Event{
		EventID:     "evt-2",
		EventType:   application.EventTypeOrderCreated,
		AggregateID: "11111111-1111-4111-8111-111111111111",
		Payload:     json.RawMessage(`{"status":"created"}`),
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := DispatchToHub(hub, body); err == nil {
		t.Fatal("expected error")
	}
	if len(hub.events) != 0 {
		t.Fatalf("published %d, want 0", len(hub.events))
	}
}

func TestConsume_duplicateEventIDPublishesOnce(t *testing.T) {
	msg := hubMessage(t, "evt-1")
	hub := &recordingHub{}
	c := &Consumer{hub: hub, seen: newEventSeen(10)}

	if err := c.consume(context.Background(), msg); err != nil {
		t.Fatalf("first consume() error = %v", err)
	}
	if err := c.consume(context.Background(), msg); err != nil {
		t.Fatalf("second consume() error = %v", err)
	}
	if len(hub.events) != 1 {
		t.Fatalf("hub events = %d, want 1", len(hub.events))
	}
}

func TestConsume_differentEventIDsBothPublish(t *testing.T) {
	hub := &recordingHub{}
	c := &Consumer{hub: hub, seen: newEventSeen(10)}
	if err := c.consume(context.Background(), hubMessage(t, "evt-1")); err != nil {
		t.Fatalf("consume evt-1: %v", err)
	}
	if err := c.consume(context.Background(), hubMessage(t, "evt-2")); err != nil {
		t.Fatalf("consume evt-2: %v", err)
	}
	if len(hub.events) != 2 {
		t.Fatalf("hub events = %d, want 2", len(hub.events))
	}
}

func TestConsume_parseErrorIsNotSuccess(t *testing.T) {
	body, err := json.Marshal(Event{
		EventID:     "evt-bad",
		EventType:   application.EventTypeOrderCreated,
		AggregateID: "11111111-1111-4111-8111-111111111111",
		Payload:     json.RawMessage(`{"status":"created"}`),
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	c := &Consumer{hub: &recordingHub{}, seen: newEventSeen(10)}
	msg := kafkago.Message{Topic: "order.events", Partition: 0, Offset: 4, Value: body}

	if err := c.consume(context.Background(), msg); err == nil {
		t.Fatal("consume() error = nil, want parse error")
	}
	if err := c.consume(context.Background(), msg); err == nil {
		t.Fatal("retry consume() error = nil, parse error must not be stored as seen")
	}
	if c.seen.has("evt-bad") {
		t.Fatal("failed event_id was remembered")
	}
}

func TestApply_errorDoesNotReturnUntilCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Consumer{hub: &recordingHub{}, seen: newEventSeen(10)}
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	err := c.apply(ctx, kafkago.Message{
		Topic:     "order.events",
		Partition: 1,
		Offset:    9,
		Value:     []byte(`{`),
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("apply() error = %v, want context.Canceled", err)
	}
	if len(c.hub.(*recordingHub).events) != 0 {
		t.Fatal("invalid event was published")
	}
}

func TestEventSeen_dropsOldest(t *testing.T) {
	seen := newEventSeen(1)
	seen.mark("evt-1")
	seen.mark("evt-2")
	if seen.has("evt-1") {
		t.Fatal("oldest event_id was kept")
	}
	if !seen.has("evt-2") {
		t.Fatal("newest event_id was dropped")
	}
}

func hubMessage(t *testing.T, eventID string) kafkago.Message {
	t.Helper()
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	orderID := "11111111-1111-4111-8111-111111111111"
	payload, err := json.Marshal(map[string]any{
		"order_id":   orderID,
		"user_id":    "22222222-2222-4222-8222-222222222222",
		"market_id":  "BTC-USDT",
		"status":     "filled",
		"updated_at": now,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	msg, err := messageFromEvent(context.Background(), application.OutboxEvent{
		ID:          eventID,
		AggregateID: orderID,
		EventType:   application.EventTypeOrderStatusChanged,
		Payload:     payload,
		CreatedAt:   now,
	})
	if err != nil {
		t.Fatalf("messageFromEvent() error = %v", err)
	}
	return msg
}

func TestHubForwarder_eachReplicaReceivesOnce(t *testing.T) {
	msg := hubMessage(t, "evt-1")
	first := &recordingHub{}
	second := &recordingHub{}
	forwardFirst := NewHubForwarder(first)
	forwardSecond := NewHubForwarder(second)

	if err := forwardFirst.Deliver(msg.Value); err != nil {
		t.Fatalf("first Deliver() error = %v", err)
	}
	if err := forwardFirst.Deliver(msg.Value); err != nil {
		t.Fatalf("first duplicate Deliver() error = %v", err)
	}
	if err := forwardSecond.Deliver(msg.Value); err != nil {
		t.Fatalf("second Deliver() error = %v", err)
	}
	if len(first.events) != 1 || len(second.events) != 1 {
		t.Fatalf("hub events = %d and %d, want 1 and 1", len(first.events), len(second.events))
	}
}

func TestNewHubConsumerGroupID_unique(t *testing.T) {
	a := NewHubConsumerGroupID()
	b := NewHubConsumerGroupID()
	if a == b {
		t.Fatal("group ids must be unique per process")
	}
	if len(a) < len("orderservice-hub-") {
		t.Fatalf("group id = %q", a)
	}
}
