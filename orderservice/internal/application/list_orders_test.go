package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/memory"
)

func TestListOrders_returnsUserOrders(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := application.NewCreateOrder(repo, marketCheckerStub{}, nil, nil, nil)
	list := application.NewListOrders(repo)

	now := time.Now().UTC()
	for i, marketID := range []string{"BTC-USDT", "ETH-USDT"} {
		order, err := domain.NewOrder(
			domain.NewOrderID(),
			"user-1",
			marketID,
			domain.OrderSideBuy,
			domain.Money{},
			mustDecimal(t, "0.1"),
			now.Add(time.Duration(i)*time.Second),
		)
		if err != nil {
			t.Fatalf("NewOrder() error = %v", err)
		}
		if err := repo.Create(context.Background(), order); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	_ = uc

	out, err := list.Execute(context.Background(), application.ListOrdersInput{UserID: "user-1"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(out.Orders) != 2 {
		t.Fatalf("orders count = %d, want 2", len(out.Orders))
	}
}
