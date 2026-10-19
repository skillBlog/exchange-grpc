package fanout

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
)

type stubPublisher struct {
	err   error
	calls int
}

func (s *stubPublisher) Publish(context.Context, application.OutboxEvent) error {
	s.calls++
	return s.err
}

type recordingHub struct {
	events []application.UpdateEvent
}

func (h *recordingHub) Publish(event application.UpdateEvent) {
	h.events = append(h.events, event)
}

func TestChain_stopsOnFirstError(t *testing.T) {
	first := &stubPublisher{err: errors.New("kafka down")}
	second := &stubPublisher{}
	err := New(first, second).Publish(context.Background(), application.OutboxEvent{ID: "evt-1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if first.calls != 1 || second.calls != 0 {
		t.Fatalf("calls = %d %d, want 1 0", first.calls, second.calls)
	}
}

func TestChain_callsBoth(t *testing.T) {
	first := &stubPublisher{}
	second := &stubPublisher{}
	if err := New(first, second).Publish(context.Background(), application.OutboxEvent{ID: "evt-1"}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if first.calls != 1 || second.calls != 1 {
		t.Fatalf("calls = %d %d, want 1 1", first.calls, second.calls)
	}
}

func TestHubPublisher_duplicateEventIDPublishesOnce(t *testing.T) {
	hub := &recordingHub{}
	publisher := NewHubPublisher(hub)
	event := testEvent(t)
	if err := publisher.Publish(context.Background(), event); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if err := publisher.Publish(context.Background(), event); err != nil {
		t.Fatalf("second Publish() error = %v", err)
	}
	if len(hub.events) != 1 {
		t.Fatalf("hub events = %d, want 1", len(hub.events))
	}
	if hub.events[0].OrderID != event.AggregateID || hub.events[0].Status != domain.OrderStatusFilled {
		t.Fatalf("event = %+v", hub.events[0])
	}
}

func testEvent(t *testing.T) application.OutboxEvent {
	t.Helper()
	orderID := "11111111-1111-4111-8111-111111111111"
	payload, err := json.Marshal(map[string]any{
		"order_id":   orderID,
		"user_id":    "22222222-2222-4222-8222-222222222222",
		"market_id":  "BTC-USDT",
		"status":     "filled",
		"updated_at": time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return application.OutboxEvent{
		ID:          "evt-1",
		AggregateID: orderID,
		EventType:   application.EventTypeOrderStatusChanged,
		Payload:     payload,
		CreatedAt:   time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC),
	}
}
