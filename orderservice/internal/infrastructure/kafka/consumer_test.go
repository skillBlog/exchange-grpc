package kafka

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
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

	msg, err := messageFromEvent(application.OutboxEvent{
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
