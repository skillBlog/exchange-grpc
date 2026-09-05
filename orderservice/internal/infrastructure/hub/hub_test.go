package hub_test

import (
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/hub"
	"github.com/exchange-grpc/shared/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestUpdateHub_doubleUnsubscribeDoesNotPanic(t *testing.T) {
	orderHub := hub.NewUpdateHub(4, logger.NewNop(), 0)

	_, unsubscribe := orderHub.Subscribe("order-1")
	unsubscribe()
	unsubscribe()

	_, unsubUser := orderHub.SubscribeUser("user-1")
	unsubUser()
	unsubUser()
}

func TestUpdateHub_publishTimeoutIsLogged(t *testing.T) {
	core, observed := observer.New(zap.WarnLevel)
	orderHub := hub.NewUpdateHub(1, zap.New(core), 20*time.Millisecond)

	_, unsubscribe := orderHub.Subscribe("order-1")
	defer unsubscribe()

	now := time.Now().UTC()
	orderHub.Publish(application.UpdateEvent{OrderID: "order-1", Status: domain.OrderStatusCreated, UpdatedAt: now})
	orderHub.Publish(application.UpdateEvent{OrderID: "order-1", Status: domain.OrderStatusFilled, UpdatedAt: now})

	if observed.Len() != 1 {
		t.Fatalf("log entries = %d, want 1", observed.Len())
	}
	entry := observed.All()[0]
	if entry.Message != "order update timed out waiting for subscriber" {
		t.Fatalf("message = %q", entry.Message)
	}
}

func TestUpdateHub_subscriberReceivesPublishedEvents(t *testing.T) {
	orderHub := hub.NewUpdateHub(4, logger.NewNop(), 0)

	updates, unsubscribe := orderHub.Subscribe("order-1")
	defer unsubscribe()

	orderHub.Publish(application.UpdateEvent{OrderID: "order-1", Status: domain.OrderStatusFilled, UpdatedAt: time.Now().UTC()})

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

func TestUpdateHub_userSubscriberReceivesEvents(t *testing.T) {
	orderHub := hub.NewUpdateHub(4, logger.NewNop(), 0)

	updates, unsubscribe := orderHub.SubscribeUser("user-1")
	defer unsubscribe()
	other, unsubOther := orderHub.SubscribeUser("user-2")
	defer unsubOther()

	orderHub.Publish(application.UpdateEvent{
		OrderID:   "order-1",
		UserID:    "user-1",
		Status:    domain.OrderStatusCreated,
		UpdatedAt: time.Now().UTC(),
	})

	select {
	case event := <-updates:
		if event.OrderID != "order-1" {
			t.Fatalf("OrderID = %q", event.OrderID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for user event")
	}

	select {
	case event := <-other:
		t.Fatalf("other user received %+v", event)
	default:
	}
}

func TestUpdateHub_publishFansOutToOrderAndUser(t *testing.T) {
	orderHub := hub.NewUpdateHub(4, logger.NewNop(), 0)

	byOrder, unsubOrder := orderHub.Subscribe("order-1")
	defer unsubOrder()
	byUser, unsubUser := orderHub.SubscribeUser("user-1")
	defer unsubUser()

	orderHub.Publish(application.UpdateEvent{
		OrderID:   "order-1",
		UserID:    "user-1",
		Status:    domain.OrderStatusFilled,
		UpdatedAt: time.Now().UTC(),
	})

	for _, ch := range []<-chan application.UpdateEvent{byOrder, byUser} {
		select {
		case event := <-ch:
			if event.Status != domain.OrderStatusFilled {
				t.Fatalf("Status = %q", event.Status)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for fan-out event")
		}
	}
}
