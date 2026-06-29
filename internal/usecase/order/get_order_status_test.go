package order_test

import (
	"context"
	"errors"
	"testing"

	"github.com/exchange-grpc/internal/adapter/repository/memory"
	"github.com/exchange-grpc/internal/domain"
	"github.com/exchange-grpc/internal/usecase/order"
)

func TestGetOrderStatus_success(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := order.NewGetOrderStatus(repo)

	existing, err := domain.NewOrder("order-1", "user-1", "BTC-USDT", domain.OrderTypeLimit, "100", "0.1")
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	if err := repo.Save(context.Background(), existing); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := uc.Execute(context.Background(), order.GetOrderStatusInput{
		OrderID: "order-1",
		UserID:  "user-1",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.ID != existing.ID {
		t.Fatalf("ID = %q, want %q", got.ID, existing.ID)
	}
}

func TestGetOrderStatus_notFound(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := order.NewGetOrderStatus(repo)

	existing, err := domain.NewOrder("order-1", "user-1", "BTC-USDT", domain.OrderTypeMarket, "", "1")
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	if err := repo.Save(context.Background(), existing); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	_, err = uc.Execute(context.Background(), order.GetOrderStatusInput{
		OrderID: "order-1",
		UserID:  "user-2",
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestGetOrderStatus_invalidInput(t *testing.T) {
	uc := order.NewGetOrderStatus(memory.NewOrderRepository())

	_, err := uc.Execute(context.Background(), order.GetOrderStatusInput{
		UserID: "user-1",
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}

	_, err = uc.Execute(context.Background(), order.GetOrderStatusInput{
		OrderID: "order-1",
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}
