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

func mustMoney(t *testing.T, amount string) domain.Money {
	t.Helper()
	money, err := domain.NewMoney(amount, "USD")
	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}
	return money
}

func mustDecimal(t *testing.T, value string) domain.Decimal {
	t.Helper()
	decimal, err := domain.NewDecimal(value)
	if err != nil {
		t.Fatalf("NewDecimal() error = %v", err)
	}
	return decimal
}

func TestCreateOrder_success(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := application.NewCreateOrder(repo, marketCheckerStub{}, nil, nil, nil)

	out, err := uc.Execute(context.Background(), application.CreateOrderInput{
		UserID:   "user-1",
		MarketID: "BTC-USDT",
		Side:     domain.OrderSideBuy,
		Price:    mustMoney(t, "100"),
		Quantity: mustDecimal(t, "0.1"),
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

func TestCreateOrder_idempotency(t *testing.T) {
	repo := memory.NewOrderRepository()
	idempotency := memory.NewIdempotencyStore()
	uc := application.NewCreateOrder(repo, marketCheckerStub{}, idempotency, nil, nil)

	input := application.CreateOrderInput{
		UserID:         "user-1",
		MarketID:       "BTC-USDT",
		Side:           domain.OrderSideBuy,
		Quantity:       mustDecimal(t, "0.1"),
		IdempotencyKey: "key-1",
	}

	first, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("first Execute() error = %v", err)
	}

	second, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("second Execute() error = %v", err)
	}
	if first.OrderID != second.OrderID {
		t.Fatalf("order ids differ: %s vs %s", first.OrderID, second.OrderID)
	}
}

func TestCreateOrder_marketInactive(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := application.NewCreateOrder(repo, marketCheckerStub{err: domain.ErrMarketInactive}, nil, nil, nil)

	_, err := uc.Execute(context.Background(), application.CreateOrderInput{
		UserID:   "user-1",
		MarketID: "SOL-USDT",
		Side:     domain.OrderSideBuy,
		Quantity: mustDecimal(t, "1"),
	})
	if !errors.Is(err, domain.ErrMarketInactive) {
		t.Fatalf("error = %v, want ErrMarketInactive", err)
	}
}

func TestCreateOrder_forbiddenMarket(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := application.NewCreateOrder(repo, marketCheckerStub{err: domain.ErrForbidden}, nil, nil, nil)

	_, err := uc.Execute(context.Background(), application.CreateOrderInput{
		UserID:   "user-1",
		MarketID: "BNB-USDT",
		Side:     domain.OrderSideBuy,
		Quantity: mustDecimal(t, "1"),
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestCreateOrder_invalidInput(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := application.NewCreateOrder(repo, marketCheckerStub{}, nil, nil, nil)

	_, err := uc.Execute(context.Background(), application.CreateOrderInput{
		MarketID: "BTC-USDT",
		Side:     domain.OrderSideBuy,
		Price:    mustMoney(t, "100"),
		Quantity: mustDecimal(t, "0.1"),
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}
