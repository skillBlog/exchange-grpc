package order_test

import (
	"context"
	"errors"
	"testing"

	"github.com/exchange-grpc/internal/adapter/repository/memory"
	"github.com/exchange-grpc/internal/domain"
	"github.com/exchange-grpc/internal/usecase/order"
)

type marketCheckerStub struct {
	err error
}

func (s marketCheckerStub) EnsureMarketAvailable(context.Context, string, []string) error {
	return s.err
}

func TestCreateOrder_success(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := order.NewCreateOrder(repo, marketCheckerStub{}, nil)

	out, err := uc.Execute(context.Background(), order.CreateOrderInput{
		UserID:   "user-1",
		MarketID: "BTC-USDT",
		Type:     domain.OrderTypeLimit,
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
	uc := order.NewCreateOrder(repo, marketCheckerStub{
		err: domain.ErrMarketInactive,
	}, nil)

	_, err := uc.Execute(context.Background(), order.CreateOrderInput{
		UserID:   "user-1",
		MarketID: "SOL-USDT",
		Type:     domain.OrderTypeMarket,
		Quantity: "1",
	})
	if !errors.Is(err, domain.ErrMarketInactive) {
		t.Fatalf("error = %v, want ErrMarketInactive", err)
	}
}

func TestCreateOrder_marketUnavailable(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := order.NewCreateOrder(repo, marketCheckerStub{
		err: domain.ErrNotFound,
	}, nil)

	_, err := uc.Execute(context.Background(), order.CreateOrderInput{
		UserID:   "user-1",
		MarketID: "UNKNOWN",
		Type:     domain.OrderTypeMarket,
		Quantity: "1",
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}

	_, getErr := repo.GetByID(context.Background(), "any")
	if !errors.Is(getErr, domain.ErrNotFound) {
		t.Fatalf("order should not be saved, GetByID error = %v", getErr)
	}
}

func TestCreateOrder_forbiddenMarket(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := order.NewCreateOrder(repo, marketCheckerStub{
		err: domain.ErrForbidden,
	}, nil)

	_, err := uc.Execute(context.Background(), order.CreateOrderInput{
		UserID:   "user-1",
		MarketID: "BNB-USDT",
		Type:     domain.OrderTypeMarket,
		Quantity: "1",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestCreateOrder_invalidInput(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := order.NewCreateOrder(repo, marketCheckerStub{}, nil)

	_, err := uc.Execute(context.Background(), order.CreateOrderInput{
		MarketID: "BTC-USDT",
		Type:     domain.OrderTypeLimit,
		Price:    "100",
		Quantity: "0.1",
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}

func TestCreateOrder_marketCheckerCalledBeforeValidation(t *testing.T) {
	var checked bool
	checker := marketCheckerFunc(func(ctx context.Context, marketID string) error {
		checked = true
		return domain.ErrNotFound
	})

	uc := order.NewCreateOrder(memory.NewOrderRepository(), checker, nil)
	_, err := uc.Execute(context.Background(), order.CreateOrderInput{
		UserID:   "",
		MarketID: "BTC-USDT",
		Type:     domain.OrderTypeMarket,
		Quantity: "1",
	})
	if !checked {
		t.Fatal("market checker was not called")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

type marketCheckerFunc func(context.Context, string) error

func (f marketCheckerFunc) EnsureMarketAvailable(ctx context.Context, marketID string, _ []string) error {
	return f(ctx, marketID)
}
