package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/hub"
	"github.com/exchange-grpc/shared/logger"
)

const testUserID = "11111111-1111-1111-1111-111111111111"

func TestStreamUserOrderUpdates_receivesLiveEvents(t *testing.T) {
	orderHub := hub.NewUpdateHub(4, logger.NewNop(), 0)
	uc := application.NewStreamUserOrderUpdates(orderHub)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	got := make(chan application.UpdateEvent, 2)
	errCh := make(chan error, 1)
	go func() {
		errCh <- uc.Execute(ctx, application.StreamUserOrderUpdatesInput{UserID: testUserID}, func(update application.UpdateEvent) error {
			got <- update
			return nil
		})
	}()

	time.Sleep(20 * time.Millisecond)
	orderHub.Publish(application.UpdateEvent{
		OrderID:   "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		UserID:    testUserID,
		MarketID:  "BTC-USDT",
		Status:    domain.OrderStatusCreated,
		UpdatedAt: time.Now().UTC(),
	})
	orderHub.Publish(application.UpdateEvent{
		OrderID:   "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		UserID:    "22222222-2222-2222-2222-222222222222",
		MarketID:  "ETH-USDT",
		Status:    domain.OrderStatusCreated,
		UpdatedAt: time.Now().UTC(),
	})

	select {
	case event := <-got:
		if event.OrderID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
			t.Fatalf("OrderID = %q", event.OrderID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for user event")
	}

	select {
	case event := <-got:
		t.Fatalf("received other user's event %+v", event)
	case <-time.After(50 * time.Millisecond):
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("Execute() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Execute did not return after cancel")
	}
}

func TestStreamUserOrderUpdates_filtersByMarket(t *testing.T) {
	orderHub := hub.NewUpdateHub(4, logger.NewNop(), 0)
	uc := application.NewStreamUserOrderUpdates(orderHub)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	got := make(chan application.UpdateEvent, 2)
	go func() {
		_ = uc.Execute(ctx, application.StreamUserOrderUpdatesInput{
			UserID:   testUserID,
			MarketID: "BTC-USDT",
		}, func(update application.UpdateEvent) error {
			got <- update
			return nil
		})
	}()

	time.Sleep(20 * time.Millisecond)
	orderHub.Publish(application.UpdateEvent{
		OrderID:  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		UserID:   testUserID,
		MarketID: "ETH-USDT",
		Status:   domain.OrderStatusCreated,
	})
	orderHub.Publish(application.UpdateEvent{
		OrderID:  "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		UserID:   testUserID,
		MarketID: "BTC-USDT",
		Status:   domain.OrderStatusCreated,
	})

	select {
	case event := <-got:
		if event.MarketID != "BTC-USDT" {
			t.Fatalf("MarketID = %q, want BTC-USDT", event.MarketID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for filtered event")
	}

	select {
	case event := <-got:
		t.Fatalf("received unfiltered event %+v", event)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestStreamUserOrderUpdates_sendFailureUnsubscribes(t *testing.T) {
	orderHub := hub.NewUpdateHub(4, logger.NewNop(), 0)
	uc := application.NewStreamUserOrderUpdates(orderHub)

	sendErr := errors.New("send failed")
	errCh := make(chan error, 1)
	go func() {
		errCh <- uc.Execute(context.Background(), application.StreamUserOrderUpdatesInput{UserID: testUserID}, func(application.UpdateEvent) error {
			return sendErr
		})
	}()

	time.Sleep(20 * time.Millisecond)
	orderHub.Publish(application.UpdateEvent{
		OrderID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		UserID:  testUserID,
		Status:  domain.OrderStatusCreated,
	})

	select {
	case err := <-errCh:
		if !errors.Is(err, sendErr) {
			t.Fatalf("error = %v, want %v", err, sendErr)
		}
	case <-time.After(time.Second):
		t.Fatal("Execute did not return after send failure")
	}

	orderHub.Publish(application.UpdateEvent{
		OrderID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		UserID:  testUserID,
		Status:  domain.OrderStatusFilled,
	})
}

func TestStreamUserOrderUpdates_nilHubReturnsImmediately(t *testing.T) {
	uc := application.NewStreamUserOrderUpdates(nil)

	done := make(chan error, 1)
	go func() {
		done <- uc.Execute(context.Background(), application.StreamUserOrderUpdatesInput{UserID: testUserID}, func(application.UpdateEvent) error {
			return nil
		})
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error when hub is not configured")
		}
		if !strings.Contains(err.Error(), "order update hub is not configured") {
			t.Fatalf("error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Execute hung on nil hub")
	}
}

func TestStreamUserOrderUpdates_requiresUserID(t *testing.T) {
	uc := application.NewStreamUserOrderUpdates(hub.NewUpdateHub(4, logger.NewNop(), 0))
	err := uc.Execute(context.Background(), application.StreamUserOrderUpdatesInput{}, func(application.UpdateEvent) error {
		return nil
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}
