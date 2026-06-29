package order_test

import (
	"context"
	"testing"
	"time"

	"github.com/exchange-grpc/internal/adapter/repository/memory"
	"github.com/exchange-grpc/internal/domain"
	"github.com/exchange-grpc/internal/usecase/order"
)

func TestStreamOrderUpdates_receivesPublishedUpdates(t *testing.T) {
	repo := memory.NewOrderRepository()
	hub := order.NewUpdateHub()
	streamUC := order.NewStreamOrderUpdates(repo, hub)
	updateUC := order.NewUpdateOrderStatus(repo, hub)

	existing, err := domain.NewOrder("order-1", "user-1", "BTC-USDT", domain.OrderTypeMarket, "", "1")
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	if err := repo.Save(context.Background(), existing); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	events := make(chan order.UpdateEvent, 4)
	errCh := make(chan error, 1)
	go func() {
		errCh <- streamUC.Execute(ctx, order.StreamOrderUpdatesInput{
			OrderID: "order-1",
			UserID:  "user-1",
		}, func(event order.UpdateEvent) error {
			events <- event
			return nil
		})
	}()

	waitForStatus(t, events, "created")

	if err := updateUC.Execute(context.Background(), order.UpdateOrderStatusInput{
		OrderID: "order-1",
		UserID:  "user-1",
		Status:  domain.OrderStatusFilled,
	}); err != nil {
		t.Fatalf("UpdateOrderStatus() error = %v", err)
	}

	waitForStatus(t, events, "filled")
	cancel()
}

func waitForStatus(t *testing.T, events <-chan order.UpdateEvent, want string) {
	t.Helper()

	select {
	case event := <-events:
		if string(event.Status) != want {
			t.Fatalf("status = %q, want %q", event.Status, want)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for status %q", want)
	}
}
