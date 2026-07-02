package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/memory"
)

type marketCheckerStub struct {
	err error
}

func (s marketCheckerStub) EnsureMarketAvailable(context.Context, string, []string) error {
	return s.err
}

func TestCreateOrder_success(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := application.NewCreateOrder(repo, marketCheckerStub{}, nil)

	out, err := uc.Execute(context.Background(), application.CreateOrderInput{
		UserID:   "user-1",
		MarketID: "BTC-USDT",
		Side:     domain.OrderSideBuy,
		Price:    "100",
		Quantity: "0.1",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if out.OrderID == "" {
		t.Fatal("expected non-empty order id")
	}
	if out.Status != domain.OrderStatusCreated {
		t.Fatalf("Status = %q, want %q", out.Status, domain.OrderStatusCreated)
	}

	saved, err := repo.GetByID(context.Background(), out.OrderID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if saved.MarketID != "BTC-USDT" {
		t.Fatalf("MarketID = %q, want BTC-USDT", saved.MarketID)
	}
}

func TestCreateOrder_marketInactive(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := application.NewCreateOrder(repo, marketCheckerStub{err: domain.ErrMarketInactive}, nil)

	_, err := uc.Execute(context.Background(), application.CreateOrderInput{
		UserID:   "user-1",
		MarketID: "SOL-USDT",
		Side:     domain.OrderSideBuy,
		Quantity: "1",
	})
	if !errors.Is(err, domain.ErrMarketInactive) {
		t.Fatalf("error = %v, want ErrMarketInactive", err)
	}
}

func TestCreateOrder_forbiddenMarket(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := application.NewCreateOrder(repo, marketCheckerStub{err: domain.ErrForbidden}, nil)

	_, err := uc.Execute(context.Background(), application.CreateOrderInput{
		UserID:   "user-1",
		MarketID: "BNB-USDT",
		Side:     domain.OrderSideBuy,
		Quantity: "1",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestCreateOrder_invalidInput(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := application.NewCreateOrder(repo, marketCheckerStub{}, nil)

	_, err := uc.Execute(context.Background(), application.CreateOrderInput{
		MarketID: "BTC-USDT",
		Side:     domain.OrderSideBuy,
		Price:    "100",
		Quantity: "0.1",
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}
