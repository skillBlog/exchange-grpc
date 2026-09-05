package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/memory"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

type marketCheckerStub struct {
	err    error
	limits domain.MarketLimits
}

func (s marketCheckerStub) EnsureMarketAvailable(context.Context, string, []string) (domain.MarketLimits, error) {
	return s.limits, s.err
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
	uc := application.NewCreateOrder(repo, marketCheckerStub{}, nil, nil, nil, nil)

	out, err := uc.Execute(context.Background(), application.CreateOrderInput{
		UserID:   "11111111-1111-1111-1111-111111111111",
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
	uc := application.NewCreateOrder(repo, marketCheckerStub{}, idempotency, nil, nil, nil)

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

	all, err := repo.ListByUserID(context.Background(), "11111111-1111-1111-1111-111111111111", 100, "", domain.ListOrdersFilter{})
	if err != nil {
		t.Fatalf("ListByUserID() error = %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("orders count = %d, want 1 (no duplicates)", len(all))
	}
}

func TestCreateOrder_inFlightIdempotencyKey(t *testing.T) {
	repo := memory.NewOrderRepository()
	idempotency := memory.NewIdempotencyStore()
	uc := application.NewCreateOrder(repo, marketCheckerStub{}, idempotency, nil, nil, nil)

	const (
		userID = "11111111-1111-1111-1111-111111111111"
		key    = "key-in-flight"
	)
	if _, _, err := idempotency.Reserve(context.Background(), userID, key, "22222222-2222-2222-2222-222222222222"); err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}

	_, err := uc.Execute(context.Background(), application.CreateOrderInput{
		UserID:         userID,
		MarketID:       "BTC-USDT",
		Side:           domain.OrderSideBuy,
		Quantity:       mustDecimal(t, "0.1"),
		IdempotencyKey: key,
	})
	if !errors.Is(err, domain.ErrFailedPrecondition) {
		t.Fatalf("error = %v, want ErrFailedPrecondition", err)
	}

	all, err := repo.ListByUserID(context.Background(), userID, 100, "", domain.ListOrdersFilter{})
	if err != nil {
		t.Fatalf("ListByUserID() error = %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("orders count = %d, want 0 (no duplicate while key is reserved)", len(all))
	}
}

type createFailRepo struct {
	*memory.OrderRepository
	failOnce bool
}

func (r *createFailRepo) Create(ctx context.Context, order domain.Order) error {
	if r.failOnce {
		r.failOnce = false
		return errors.New("db unavailable")
	}
	return r.OrderRepository.Create(ctx, order)
}

func TestCreateOrder_marksIdempotencyFailedWhenCreateFails(t *testing.T) {
	base := memory.NewOrderRepository()
	repo := &createFailRepo{OrderRepository: base, failOnce: true}
	idempotency := memory.NewIdempotencyStore()
	uc := application.NewCreateOrder(repo, marketCheckerStub{}, idempotency, nil, nil, nil)

	input := application.CreateOrderInput{
		UserID:         "11111111-1111-1111-1111-111111111111",
		MarketID:       "BTC-USDT",
		Side:           domain.OrderSideBuy,
		Quantity:       mustDecimal(t, "0.1"),
		IdempotencyKey: "key-fail",
	}

	if _, err := uc.Execute(context.Background(), input); err == nil {
		t.Fatal("expected create error")
	}

	out, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("retry Execute() error = %v", err)
	}
	if out.OrderID == "" {
		t.Fatal("expected order id on retry after failed key rewrite")
	}
}

func TestCreateOrder_marketInactive(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := application.NewCreateOrder(repo, marketCheckerStub{err: domain.ErrMarketInactive}, nil, nil, nil, nil)

	_, err := uc.Execute(context.Background(), application.CreateOrderInput{
		UserID:   "11111111-1111-1111-1111-111111111111",
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
	uc := application.NewCreateOrder(repo, marketCheckerStub{err: domain.ErrForbidden}, nil, nil, nil, nil)

	_, err := uc.Execute(context.Background(), application.CreateOrderInput{
		UserID:   "11111111-1111-1111-1111-111111111111",
		MarketID: "BNB-USDT",
		Side:     domain.OrderSideBuy,
		Quantity: mustDecimal(t, "1"),
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

type completeFailStore struct {
	inner domain.IdempotencyStore
}

func (s completeFailStore) GetOrderID(ctx context.Context, userID, key string) (string, bool, error) {
	return s.inner.GetOrderID(ctx, userID, key)
}

func (s completeFailStore) Reserve(ctx context.Context, userID, key, orderID string) (bool, string, error) {
	return s.inner.Reserve(ctx, userID, key, orderID)
}

func (s completeFailStore) Complete(context.Context, string, string) error {
	return errors.New("complete failed")
}

func (s completeFailStore) Fail(ctx context.Context, userID, key string) error {
	return s.inner.Fail(ctx, userID, key)
}

func TestCreateOrder_completeFailureDoesNotFailClient(t *testing.T) {
	repo := memory.NewOrderRepository()
	core, logs := observer.New(zapcore.ErrorLevel)
	uc := application.NewCreateOrder(
		repo,
		marketCheckerStub{},
		completeFailStore{inner: memory.NewIdempotencyStore()},
		nil,
		nil,
		zap.New(core),
	)

	out, err := uc.Execute(context.Background(), application.CreateOrderInput{
		UserID:         "11111111-1111-1111-1111-111111111111",
		MarketID:       "BTC-USDT",
		Side:           domain.OrderSideBuy,
		Quantity:       mustDecimal(t, "0.1"),
		IdempotencyKey: "key-complete-fail",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want success after complete failure", err)
	}
	if out.OrderID == "" {
		t.Fatal("expected order id")
	}

	saved, err := repo.GetByID(context.Background(), out.OrderID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if saved.ID != out.OrderID {
		t.Fatalf("saved id = %q, want %q", saved.ID, out.OrderID)
	}

	entries := logs.FilterMessage("idempotency complete failed after order create").All()
	if len(entries) != 1 {
		t.Fatalf("complete failure logs = %d, want 1", len(entries))
	}
	fields := entries[0].ContextMap()
	if got, _ := fields["order_id"].(string); got != out.OrderID {
		t.Fatalf("order_id = %q, want %q", got, out.OrderID)
	}
	if got, _ := fields["user_id"].(string); got != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("user_id = %q", got)
	}
	if got, _ := fields["idempotency_key"].(string); got != "key-complete-fail" {
		t.Fatalf("idempotency_key = %q", got)
	}
	if _, ok := fields["error"]; !ok {
		t.Fatal("expected error field in complete failure log")
	}
}

func TestCreateOrder_invalidInput(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := application.NewCreateOrder(repo, marketCheckerStub{}, nil, nil, nil, nil)

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

func TestCreateOrder_rejectsQuantityBelowMarketMin(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := application.NewCreateOrder(repo, marketCheckerStub{
		limits: domain.MarketLimits{MinOrderSize: "0.0001", QuantityPrecision: 8, MinNotional: "10"},
	}, nil, nil, nil, nil)

	_, err := uc.Execute(context.Background(), application.CreateOrderInput{
		UserID:   "11111111-1111-1111-1111-111111111111",
		MarketID: "BTC-USDT",
		Side:     domain.OrderSideBuy,
		Quantity: mustDecimal(t, "0.00001"),
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
	all, listErr := repo.ListByUserID(context.Background(), "11111111-1111-1111-1111-111111111111", 100, "", domain.ListOrdersFilter{})
	if listErr != nil {
		t.Fatalf("ListByUserID() error = %v", listErr)
	}
	if len(all) != 0 {
		t.Fatalf("orders count = %d, want 0", len(all))
	}
}

func TestCreateOrder_rejectsQuantityPrecision(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := application.NewCreateOrder(repo, marketCheckerStub{
		limits: domain.MarketLimits{MinOrderSize: "0.0001", QuantityPrecision: 8},
	}, nil, nil, nil, nil)

	_, err := uc.Execute(context.Background(), application.CreateOrderInput{
		UserID:   "11111111-1111-1111-1111-111111111111",
		MarketID: "BTC-USDT",
		Side:     domain.OrderSideBuy,
		Quantity: mustDecimal(t, "0.000100001"),
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}

func TestCreateOrder_rejectsNotionalBelowMin(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := application.NewCreateOrder(repo, marketCheckerStub{
		limits: domain.MarketLimits{MinOrderSize: "0.0001", QuantityPrecision: 8, MinNotional: "10"},
	}, nil, nil, nil, nil)

	_, err := uc.Execute(context.Background(), application.CreateOrderInput{
		UserID:   "11111111-1111-1111-1111-111111111111",
		MarketID: "BTC-USDT",
		Side:     domain.OrderSideBuy,
		Price:    mustMoney(t, "100"),
		Quantity: mustDecimal(t, "0.01"),
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}

func TestCreateOrder_acceptsQuantityAtMarketLimits(t *testing.T) {
	repo := memory.NewOrderRepository()
	uc := application.NewCreateOrder(repo, marketCheckerStub{
		limits: domain.MarketLimits{MinOrderSize: "0.0001", QuantityPrecision: 8, MinNotional: "10"},
	}, nil, nil, nil, nil)

	out, err := uc.Execute(context.Background(), application.CreateOrderInput{
		UserID:   "11111111-1111-1111-1111-111111111111",
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
}
