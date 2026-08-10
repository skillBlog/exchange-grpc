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
	list := application.NewListOrders(repo)

	now := time.Now().UTC()
	for i, marketID := range []string{"BTC-USDT", "ETH-USDT"} {
		order, err := domain.NewOrder(
			domain.NewOrderID(),
			"11111111-1111-1111-1111-111111111111",
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

	out, err := list.Execute(context.Background(), application.ListOrdersInput{UserID: "11111111-1111-1111-1111-111111111111"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(out.Orders) != 2 {
		t.Fatalf("orders count = %d, want 2", len(out.Orders))
	}
}

func TestListOrders_paginatesWithLimit(t *testing.T) {
	repo := memory.NewOrderRepository()
	list := application.NewListOrders(repo)
	now := time.Now().UTC()

	ids := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		order, err := domain.NewOrder(
			domain.NewOrderID(),
			"11111111-1111-1111-1111-111111111111",
			"BTC-USDT",
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
		ids = append(ids, order.ID)
	}

	page1, err := list.Execute(context.Background(), application.ListOrdersInput{
		UserID:   "11111111-1111-1111-1111-111111111111",
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("page1 error = %v", err)
	}
	if len(page1.Orders) != 2 {
		t.Fatalf("page1 count = %d, want 2", len(page1.Orders))
	}
	if !page1.HasMore || page1.NextPageToken == "" {
		t.Fatal("expected has_more and next_page_token on page1")
	}

	page2, err := list.Execute(context.Background(), application.ListOrdersInput{
		UserID:    "11111111-1111-1111-1111-111111111111",
		PageSize:  2,
		PageToken: page1.NextPageToken,
	})
	if err != nil {
		t.Fatalf("page2 error = %v", err)
	}
	if len(page2.Orders) != 1 {
		t.Fatalf("page2 count = %d, want 1", len(page2.Orders))
	}
	if page2.HasMore {
		t.Fatal("expected has_more=false on last page")
	}

	seen := map[string]struct{}{}
	for _, o := range append(page1.Orders, page2.Orders...) {
		seen[o.ID] = struct{}{}
	}
	if len(seen) != 3 {
		t.Fatalf("unique orders across pages = %d, want 3", len(seen))
	}
	_ = ids
}
