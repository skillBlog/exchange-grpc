package kafka

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/google/uuid"
)

func TestMessageFromEvent_usesOrderIDKeyAndEnvelope(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	orderID := "11111111-1111-4111-8111-111111111111"
	eventID := uuid.Must(uuid.NewV7()).String()
	payload := json.RawMessage(`{"order_id":"` + orderID + `","status":"created"}`)

	msg, err := messageFromEvent(context.Background(), application.OutboxEvent{
		ID:          eventID,
		AggregateID: orderID,
		EventType:   application.EventTypeOrderCreated,
		Payload:     payload,
		CreatedAt:   now,
	})
	if err != nil {
		t.Fatalf("messageFromEvent() error = %v", err)
	}
	if string(msg.Key) != orderID {
		t.Fatalf("key = %q, want order_id", msg.Key)
	}

	var envelope Event
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if envelope.EventID != eventID {
		t.Fatalf("event_id = %q", envelope.EventID)
	}
	if envelope.EventType != application.EventTypeOrderCreated {
		t.Fatalf("event_type = %q", envelope.EventType)
	}
	if envelope.AggregateID != orderID {
		t.Fatalf("aggregate_id = %q", envelope.AggregateID)
	}
	if string(envelope.Payload) != string(payload) {
		t.Fatalf("payload = %s", envelope.Payload)
	}
	if !envelope.CreatedAt.Equal(now) {
		t.Fatalf("created_at = %v", envelope.CreatedAt)
	}

	if len(msg.Headers) != 2 {
		t.Fatalf("headers = %d, want 2", len(msg.Headers))
	}
}

func TestPing_dialsFirstBroker(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer ln.Close()

	accepted := make(chan struct{})
	go func() {
		conn, acceptErr := ln.Accept()
		if acceptErr != nil {
			return
		}
		_ = conn.Close()
		close(accepted)
	}()

	producer := NewProducer([]string{ln.Addr().String(), "192.0.2.1:1"}, "order.events", time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := producer.Ping(ctx); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
	select {
	case <-accepted:
	case <-ctx.Done():
		t.Fatal("Ping dialed a broker other than the first")
	}
}

func TestPing_emptyBrokers(t *testing.T) {
	producer := NewProducer(nil, "order.events", time.Second)
	err := producer.Ping(context.Background())
	if err == nil {
		t.Fatal("expected error for empty brokers")
	}
}
