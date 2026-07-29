package hub_test

import (
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/hub"
	"github.com/exchange-grpc/shared/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestUpdateHub_doubleUnsubscribeDoesNotPanic(t *testing.T) {
	orderHub := hub.NewUpdateHub(4, logger.NewNop())

	_, unsubscribe := orderHub.Subscribe("order-1")
	unsubscribe()
	unsubscribe()
}

func TestUpdateHub_publishDroppedEventIsLogged(t *testing.T) {
	core, observed := observer.New(zap.WarnLevel)
	orderHub := hub.NewUpdateHub(1, zap.New(core))

	_, unsubscribe := orderHub.Subscribe("order-1")
	defer unsubscribe()

	orderHub.Publish("order-1", domain.OrderStatusCreated)
	orderHub.Publish("order-1", domain.OrderStatusFilled)

	if observed.Len() != 1 {
		t.Fatalf("log entries = %d, want 1", observed.Len())
	}
	entry := observed.All()[0]
	if entry.Message != "order update dropped: subscriber buffer full" {
		t.Fatalf("message = %q", entry.Message)
	}
}

func TestUpdateHub_subscriberReceivesPublishedEvents(t *testing.T) {
	orderHub := hub.NewUpdateHub(4, logger.NewNop())

	updates, unsubscribe := orderHub.Subscribe("order-1")
	defer unsubscribe()

	orderHub.Publish("order-1", domain.OrderStatusFilled)

	select {
	case event := <-updates:
		if event.OrderID != "order-1" {
			t.Fatalf("OrderID = %q, want order-1", event.OrderID)
		}
		if event.Status != domain.OrderStatusFilled {
			t.Fatalf("Status = %q, want filled", event.Status)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}
