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
	"github.com/exchange-grpc/orderservice/internal/infrastructure/memory"
	"github.com/exchange-grpc/shared/logger"
)

func TestStreamOrderUpdates_sendFailureUnsubscribesWithoutPanic(t *testing.T) {
	repo := memory.NewOrderRepository()
	orderHub := hub.NewUpdateHub(4, logger.NewNop(), 0)
	uc := application.NewStreamOrderUpdates(repo, orderHub)

	now := time.Now().UTC()
	order, err := domain.NewOrder(domain.NewOrderID(), "11111111-1111-1111-1111-111111111111", "BTC-USDT", domain.OrderSideBuy, domain.Money{}, mustDecimal(t, "1"), now)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	if err := repo.Create(context.Background(), order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	sendErr := errors.New("send failed")
	err = uc.Execute(context.Background(), application.StreamOrderUpdatesInput{
		OrderID: order.ID,
		UserID:  "11111111-1111-1111-1111-111111111111",
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
	orderHub.Publish(application.UpdateEvent{OrderID: order.ID, Status: domain.OrderStatusFilled, UpdatedAt: time.Now().UTC()})
}

func TestStreamOrderUpdates_closesOnTerminalStatus(t *testing.T) {
	repo := memory.NewOrderRepository()
	orderHub := hub.NewUpdateHub(4, logger.NewNop(), 0)
	uc := application.NewStreamOrderUpdates(repo, orderHub)

	now := time.Now().UTC()
	order, err := domain.NewOrder(domain.NewOrderID(), "11111111-1111-1111-1111-111111111111", "BTC-USDT", domain.OrderSideBuy, domain.Money{}, mustDecimal(t, "1"), now)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	order.Status = domain.OrderStatusFilled
	if err := repo.Create(context.Background(), order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	var sent []domain.OrderStatus
	err = uc.Execute(context.Background(), application.StreamOrderUpdatesInput{
		OrderID: order.ID,
		UserID:  "11111111-1111-1111-1111-111111111111",
	}, func(update application.UpdateEvent) error {
		sent = append(sent, update.Status)
		return nil
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(sent) != 1 || sent[0] != domain.OrderStatusFilled {
		t.Fatalf("sent = %v, want [filled]", sent)
	}
}

func TestStreamOrderUpdates_nilHubReturnsImmediately(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := application.NewStreamOrderUpdates(repo, nil)

	now := time.Now().UTC()
	order, err := domain.NewOrder(domain.NewOrderID(), "11111111-1111-1111-1111-111111111111", "BTC-USDT", domain.OrderSideBuy, domain.Money{}, mustDecimal(t, "1"), now)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	if err := repo.Create(context.Background(), order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- uc.Execute(context.Background(), application.StreamOrderUpdatesInput{
			OrderID: order.ID,
			UserID:  "11111111-1111-1111-1111-111111111111",
		}, func(application.UpdateEvent) error { return nil })
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
