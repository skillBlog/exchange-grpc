package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	sharedgrpc "github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/memory"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestUpdateOrderStatus_rejectsInvalidTransition(t *testing.T) {
	repo := memory.NewOrderRepository()
	now := time.Now().UTC()
	order, err := domain.NewOrder(domain.NewOrderID(), "11111111-1111-1111-1111-111111111111", "BTC-USDT", domain.OrderSideBuy, domain.Money{}, mustDecimal(t, "1"), now)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	order.Status = domain.OrderStatusFailed
	order.UpdatedAt = now
	if err := repo.Create(context.Background(), order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	uc := application.NewUpdateOrderStatus(repo, nil, nil, nil)
	err = uc.Execute(context.Background(), application.UpdateOrderStatusInput{
		OrderID: order.ID,
		UserID:  "11111111-1111-1111-1111-111111111111",
		Status:  domain.OrderStatusCreated,
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}

func TestUpdateOrderStatus_auditIncludesRequestID(t *testing.T) {
	repo := memory.NewOrderRepository()
	now := time.Now().UTC()
	order, err := domain.NewOrder(domain.NewOrderID(), "11111111-1111-1111-1111-111111111111", "BTC-USDT", domain.OrderSideBuy, domain.Money{}, mustDecimal(t, "1"), now)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	if err := repo.Create(context.Background(), order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	core, logs := observer.New(zapcore.InfoLevel)
	uc := application.NewUpdateOrderStatus(repo, nil, nil, zap.New(core))
	ctx := sharedgrpc.ContextWithRequestID(context.Background(), "req-status-1")
	if err := uc.Execute(ctx, application.UpdateOrderStatusInput{
		OrderID: order.ID,
		UserID:  "11111111-1111-1111-1111-111111111111",
		Status:  domain.OrderStatusCancelled,
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	entries := logs.FilterMessage("order status updated").All()
	if len(entries) != 1 {
		t.Fatalf("audit logs = %d, want 1", len(entries))
	}
	fields := entries[0].ContextMap()
	if got, _ := fields["order_id"].(string); got != order.ID {
		t.Fatalf("order_id = %q", got)
	}
	if got, _ := fields["user_id"].(string); got != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("user_id = %q", got)
	}
	if got, _ := fields["request_id"].(string); got != "req-status-1" {
		t.Fatalf("request_id = %q", got)
	}
	if got, _ := fields["status"].(string); got != string(domain.OrderStatusCancelled) {
		t.Fatalf("status = %q", got)
	}
}

type recordingOutbox struct {
	events []application.OutboxEvent
}

func (o *recordingOutbox) Append(_ context.Context, event application.OutboxEvent) error {
	o.events = append(o.events, event)
	return nil
}

type failOutbox struct {
	err error
}

func (o failOutbox) Append(context.Context, application.OutboxEvent) error {
	return o.err
}

type raceStatusRepo struct {
	*memory.OrderRepository
}

func (r raceStatusRepo) UpdateStatus(ctx context.Context, id string, expected, next domain.OrderStatus, updatedAt time.Time) error {
	if err := r.OrderRepository.UpdateStatus(ctx, id, domain.OrderStatusCreated, domain.OrderStatusFilled, updatedAt); err != nil {
		return err
	}
	return r.OrderRepository.UpdateStatus(ctx, id, expected, next, updatedAt)
}

func TestUpdateOrderStatus_conflictDoesNotAppend(t *testing.T) {
	base := memory.NewOrderRepository()
	now := time.Now().UTC()
	order, err := domain.NewOrder(domain.NewOrderID(), "11111111-1111-1111-1111-111111111111", "BTC-USDT", domain.OrderSideBuy, domain.Money{}, mustDecimal(t, "1"), now)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	if err := base.Create(context.Background(), order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	outbox := &recordingOutbox{}
	uc := application.NewUpdateOrderStatus(raceStatusRepo{OrderRepository: base}, nil, outbox, nil)
	err = uc.Execute(context.Background(), application.UpdateOrderStatusInput{
		OrderID: order.ID,
		UserID:  "11111111-1111-1111-1111-111111111111",
		Status:  domain.OrderStatusCancelled,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("error = %v, want ErrConflict", err)
	}
	if len(outbox.events) != 0 {
		t.Fatalf("appended %d events, want 0", len(outbox.events))
	}

	saved, err := base.GetByID(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if saved.Status != domain.OrderStatusFilled {
		t.Fatalf("status = %q, want filled", saved.Status)
	}
}

func TestUpdateOrderStatus_sameStatusIsNoop(t *testing.T) {
	repo := memory.NewOrderRepository()
	now := time.Now().UTC()
	order, err := domain.NewOrder(domain.NewOrderID(), "11111111-1111-1111-1111-111111111111", "BTC-USDT", domain.OrderSideBuy, domain.Money{}, mustDecimal(t, "1"), now)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	if err := repo.Create(context.Background(), order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	outbox := &recordingOutbox{}
	uc := application.NewUpdateOrderStatus(repo, nil, outbox, nil)
	input := application.UpdateOrderStatusInput{
		OrderID: order.ID,
		UserID:  "11111111-1111-1111-1111-111111111111",
		Status:  domain.OrderStatusCancelled,
	}
	if err := uc.Execute(context.Background(), input); err != nil {
		t.Fatalf("first Execute() error = %v", err)
	}
	if err := uc.Execute(context.Background(), input); err != nil {
		t.Fatalf("retry Execute() error = %v", err)
	}
	if len(outbox.events) != 1 {
		t.Fatalf("appended %d events, want 1", len(outbox.events))
	}
	if outbox.events[0].EventType != application.EventTypeOrderStatusChanged {
		t.Fatalf("event_type = %q, want %q", outbox.events[0].EventType, application.EventTypeOrderStatusChanged)
	}
}

func TestUpdateOrderStatus_outboxFailureReturnsError(t *testing.T) {
	repo := memory.NewOrderRepository()
	now := time.Now().UTC()
	order, err := domain.NewOrder(domain.NewOrderID(), "11111111-1111-1111-1111-111111111111", "BTC-USDT", domain.OrderSideBuy, domain.Money{}, mustDecimal(t, "1"), now)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	if err := repo.Create(context.Background(), order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	uc := application.NewUpdateOrderStatus(repo, nil, failOutbox{err: errors.New("outbox down")}, nil)
	err = uc.Execute(context.Background(), application.UpdateOrderStatusInput{
		OrderID: order.ID,
		UserID:  "11111111-1111-1111-1111-111111111111",
		Status:  domain.OrderStatusCancelled,
	})
	if err == nil {
		t.Fatal("expected error when outbox Append fails")
	}
}
