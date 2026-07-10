package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/memory"
	"github.com/exchange-grpc/shared/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestUpdateHub_doubleUnsubscribeDoesNotPanic(t *testing.T) {
	hub := application.NewUpdateHub(4, logger.NewNop())

	_, unsubscribe := hub.Subscribe("order-1")
	unsubscribe()
	unsubscribe()
}

func TestUpdateHub_publishDroppedEventIsLogged(t *testing.T) {
	core, observed := observer.New(zap.WarnLevel)
	hub := application.NewUpdateHub(1, zap.New(core))

	_, unsubscribe := hub.Subscribe("order-1")
	defer unsubscribe()

	hub.Publish("order-1", domain.OrderStatusCreated)
	hub.Publish("order-1", domain.OrderStatusFilled)

	if observed.Len() != 1 {
		t.Fatalf("log entries = %d, want 1", observed.Len())
	}
	entry := observed.All()[0]
	if entry.Message != "order update dropped: subscriber buffer full" {
		t.Fatalf("message = %q", entry.Message)
	}
}

func TestUpdateHub_subscriberReceivesPublishedEvents(t *testing.T) {
	hub := application.NewUpdateHub(4, logger.NewNop())

	updates, unsubscribe := hub.Subscribe("order-1")
	defer unsubscribe()

	hub.Publish("order-1", domain.OrderStatusFilled)

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

func TestStreamOrderUpdates_sendErrorUnsubscribesSafely(t *testing.T) {
	repo := memory.NewOrderRepository()
	hub := application.NewUpdateHub(4, logger.NewNop())
	uc := application.NewStreamOrderUpdates(repo, hub)

	now := time.Now().UTC()
	order, err := domain.NewOrder(domain.NewOrderID(), "user-1", "BTC-USDT", domain.OrderSideBuy, domain.Money{}, mustDecimal(t, "1"), now)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	if err := repo.Create(context.Background(), order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	sendErr := errors.New("send failed")
	err = uc.Execute(context.Background(), application.StreamOrderUpdatesInput{
		OrderID: order.ID,
		UserID:  "user-1",
	}, func(update application.UpdateEvent) error {
		if update.Status == domain.OrderStatusCreated {
			return sendErr
		}
		return nil
	})
	if !errors.Is(err, sendErr) {
		t.Fatalf("error = %v, want %v", err, sendErr)
	}

	// Повторная отписка через defer не должна паниковать; hub не должен держать подписчика.
	hub.Publish(order.ID, domain.OrderStatusFilled)
}
