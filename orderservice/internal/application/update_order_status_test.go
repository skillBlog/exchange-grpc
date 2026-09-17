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

	uc := application.NewUpdateOrderStatus(repo, nil, nil)
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
	uc := application.NewUpdateOrderStatus(repo, nil, zap.New(core))
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
