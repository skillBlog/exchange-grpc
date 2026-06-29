package domain_test

import (
	"errors"
	"testing"

	"github.com/exchange-grpc/internal/domain"
)

func TestNewOrder_success(t *testing.T) {
	order, err := domain.NewOrder(
		"order-1",
		"user-1",
		"BTC-USDT",
		domain.OrderTypeLimit,
		"100.5",
		"0.01",
	)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}

	if order.ID != "order-1" {
		t.Fatalf("ID = %q, want order-1", order.ID)
	}
	if order.Status != domain.OrderStatusCreated {
		t.Fatalf("Status = %q, want %q", order.Status, domain.OrderStatusCreated)
	}
}

func TestNewOrder_validation(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		marketID  string
		orderType domain.OrderType
		price     string
		quantity  string
	}{
		{name: "missing user_id", marketID: "BTC-USDT", orderType: domain.OrderTypeMarket, quantity: "1"},
		{name: "missing market_id", userID: "user-1", orderType: domain.OrderTypeMarket, quantity: "1"},
		{name: "missing quantity", userID: "user-1", marketID: "BTC-USDT", orderType: domain.OrderTypeMarket},
		{name: "missing price for limit", userID: "user-1", marketID: "BTC-USDT", orderType: domain.OrderTypeLimit, quantity: "1"},
		{name: "unsupported type", userID: "user-1", marketID: "BTC-USDT", orderType: domain.OrderType("stop"), quantity: "1", price: "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewOrder("order-1", tt.userID, tt.marketID, tt.orderType, tt.price, tt.quantity)
			if !errors.Is(err, domain.ErrInvalidArgument) {
				t.Fatalf("error = %v, want ErrInvalidArgument", err)
			}
		})
	}
}

func TestNewOrder_marketOrderWithoutPrice(t *testing.T) {
	order, err := domain.NewOrder(
		"order-1",
		"user-1",
		"BTC-USDT",
		domain.OrderTypeMarket,
		"",
		"0.5",
	)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	if order.Type != domain.OrderTypeMarket {
		t.Fatalf("Type = %q, want market", order.Type)
	}
}
