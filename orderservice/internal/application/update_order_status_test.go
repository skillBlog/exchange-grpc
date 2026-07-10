package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/memory"
)

func TestUpdateOrderStatus_rejectsInvalidTransition(t *testing.T) {
	repo := memory.NewOrderRepository()
	now := time.Now().UTC()
	order, err := domain.NewOrder(domain.NewOrderID(), "user-1", "BTC-USDT", domain.OrderSideBuy, domain.Money{}, mustDecimal(t, "1"), now)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	order.Status = domain.OrderStatusFailed
	order.UpdatedAt = now
	if err := repo.Create(context.Background(), order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	uc := application.NewUpdateOrderStatus(repo, nil)
	err = uc.Execute(context.Background(), application.UpdateOrderStatusInput{
		OrderID: order.ID,
		UserID:  "user-1",
		Status:  domain.OrderStatusCreated,
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}
