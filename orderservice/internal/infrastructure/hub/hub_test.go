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

	_, unsubUser := orderHub.SubscribeUser("user-1", application.UserStreamFilter{})
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

	updates, unsubscribe := orderHub.SubscribeUser("user-1", application.UserStreamFilter{})
	defer unsubscribe()
	other, unsubOther := orderHub.SubscribeUser("user-2", application.UserStreamFilter{})
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
	byUser, unsubUser := orderHub.SubscribeUser("user-1", application.UserStreamFilter{})
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

func TestUpdateHub_unsubscribeThenPublishDoesNotPanic(t *testing.T) {
	core, logs := observer.New(zap.ErrorLevel)
	orderHub := hub.NewUpdateHub(4, zap.New(core), 20*time.Millisecond)

	_, unsubscribe := orderHub.Subscribe("order-1")
	unsubscribe()

	orderHub.Publish(application.UpdateEvent{
		OrderID:   "order-1",
		Status:    domain.OrderStatusFilled,
		UpdatedAt: time.Now().UTC(),
	})

	if logs.FilterMessage("order update publish panic recovered").Len() != 0 {
		t.Fatal("unsubscribe must not close the channel and panic Publish")
	}
}

func TestUpdateHub_userFilterDropsNonMatchingEvents(t *testing.T) {
	orderHub := hub.NewUpdateHub(4, logger.NewNop(), 0)

	updates, unsubscribe := orderHub.SubscribeUser("user-1", application.UserStreamFilter{
		MarketID: "BTC-USDT",
		Status:   domain.OrderStatusCreated,
	})
	defer unsubscribe()

	orderHub.Publish(application.UpdateEvent{
		OrderID:  "order-eth",
		UserID:   "user-1",
		MarketID: "ETH-USDT",
		Status:   domain.OrderStatusCreated,
	})
	orderHub.Publish(application.UpdateEvent{
		OrderID:  "order-filled",
		UserID:   "user-1",
		MarketID: "BTC-USDT",
		Status:   domain.OrderStatusFilled,
	})
	orderHub.Publish(application.UpdateEvent{
		OrderID:  "order-btc",
		UserID:   "user-1",
		MarketID: "BTC-USDT",
		Status:   domain.OrderStatusCreated,
	})

	select {
	case event := <-updates:
		if event.OrderID != "order-btc" {
			t.Fatalf("OrderID = %q, want order-btc", event.OrderID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for matching event")
	}

	select {
	case event := <-updates:
		t.Fatalf("received extra event %+v", event)
	default:
	}
}

func TestUpdateHub_concurrentUnsubscribeAndPublish(t *testing.T) {
	orderHub := hub.NewUpdateHub(8, logger.NewNop(), time.Millisecond)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			orderHub.Publish(application.UpdateEvent{
				OrderID: "order-1",
				UserID:  "user-1",
				Status:  domain.OrderStatusCreated,
			})
		}
	}()
	for i := 0; i < 200; i++ {
		_, unsub := orderHub.Subscribe("order-1")
		unsub()
		_, unsubUser := orderHub.SubscribeUser("user-1", application.UserStreamFilter{MarketID: "BTC-USDT"})
		unsubUser()
	}
	<-done
}
