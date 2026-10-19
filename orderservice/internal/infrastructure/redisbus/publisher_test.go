package redisbus

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	orderkafka "github.com/exchange-grpc/orderservice/internal/infrastructure/kafka"
)

func TestPublisher_sendsEnvelope(t *testing.T) {
	var channel string
	var body []byte
	publisher := &Publisher{
		channel: OrderUpdatesChannel,
		publish: func(_ context.Context, ch string, payload []byte) error {
			channel = ch
			body = append([]byte(nil), payload...)
			return nil
		},
	}
	event := application.OutboxEvent{
		ID:          "evt-1",
		AggregateID: "11111111-1111-4111-8111-111111111111",
		EventType:   application.EventTypeOrderCreated,
		Payload:     []byte(`{"order_id":"11111111-1111-4111-8111-111111111111"}`),
		CreatedAt:   time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC),
	}
	if err := publisher.Publish(context.Background(), event); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if channel != OrderUpdatesChannel {
		t.Fatalf("channel = %q", channel)
	}
	var envelope orderkafka.Event
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if envelope.EventID != event.ID || envelope.AggregateID != event.AggregateID {
		t.Fatalf("envelope = %+v", envelope)
	}
}

func TestPublisher_redisError(t *testing.T) {
	publisher := &Publisher{
		channel: OrderUpdatesChannel,
		publish: func(context.Context, string, []byte) error {
			return errors.New("down")
		},
	}
	err := publisher.Publish(context.Background(), application.OutboxEvent{ID: "evt-1"})
	if err == nil {
		t.Fatal("expected error")
	}
}
