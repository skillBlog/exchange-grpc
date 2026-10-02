package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/memory"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/ratelimit"
)

func TestCreateOrder_rateLimited(t *testing.T) {
	repo := memory.NewOrderRepository()
	limiter := ratelimit.NewCreateOrderLimiter(application.CreateOrderRateLimitConfig{
		GlobalLimit:  100,
		GlobalWindow: time.Minute,
		BasicLimit:   1,
		PremiumLimit: 100,
		AdminLimit:   100,
		UserWindow:   time.Minute,
	})
	uc := application.NewCreateOrder(repo, marketCheckerStub{}, nil, nil, nil, limiter, nil)

	input := application.CreateOrderInput{
		UserID:   "11111111-1111-1111-1111-111111111111",
		MarketID: "BTC-USDT",
		Side:     domain.OrderSideBuy,
		Quantity: mustDecimal(t, "0.1"),
	}

	if _, err := uc.Execute(context.Background(), input); err != nil {
		t.Fatalf("first Execute() error = %v", err)
	}

	_, err := uc.Execute(context.Background(), input)
	if !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("second error = %v, want ErrRateLimited", err)
	}
	if !strings.Contains(err.Error(), "create order rate limit") {
		t.Fatalf("error = %v, want wrap prefix", err)
	}
}

func TestCreateOrder_idempotencyBypassesRateLimit(t *testing.T) {
	repo := memory.NewOrderRepository()
	idempotency := memory.NewIdempotencyStore()
	limiter := ratelimit.NewCreateOrderLimiter(application.CreateOrderRateLimitConfig{
		GlobalLimit:  100,
		GlobalWindow: time.Minute,
		BasicLimit:   1,
		PremiumLimit: 100,
		AdminLimit:   100,
		UserWindow:   time.Minute,
	})
	uc := application.NewCreateOrder(repo, marketCheckerStub{}, idempotency, nil, nil, limiter, nil)

	input := application.CreateOrderInput{
		UserID:         "11111111-1111-1111-1111-111111111111",
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

func TestCreateOrder_marketCheckBeforeRateLimit(t *testing.T) {
	repo := memory.NewOrderRepository()
	limiter := ratelimit.NewCreateOrderLimiter(application.CreateOrderRateLimitConfig{
		GlobalLimit:  100,
		GlobalWindow: time.Minute,
		BasicLimit:   1,
		PremiumLimit: 100,
		AdminLimit:   100,
		UserWindow:   time.Minute,
	})
	uc := application.NewCreateOrder(repo, marketCheckerStub{err: domain.ErrMarketInactive}, nil, nil, nil, limiter, nil)

	input := application.CreateOrderInput{
		UserID:   "11111111-1111-1111-1111-111111111111",
		MarketID: "BTC-USDT",
		Side:     domain.OrderSideBuy,
		Quantity: mustDecimal(t, "0.1"),
	}

	_, err := uc.Execute(context.Background(), input)
	if !errors.Is(err, domain.ErrMarketInactive) {
		t.Fatalf("error = %v, want ErrMarketInactive", err)
	}

	ucOK := application.NewCreateOrder(repo, marketCheckerStub{}, nil, nil, nil, limiter, nil)
	if _, err := ucOK.Execute(context.Background(), input); err != nil {
		t.Fatalf("Execute after inactive market must not be rate-limited: %v", err)
	}
}
