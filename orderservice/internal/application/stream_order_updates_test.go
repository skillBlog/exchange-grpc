package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/hub"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/memory"
	"github.com/exchange-grpc/shared/logger"
)

func TestStreamOrderUpdates_sendFailureUnsubscribesWithoutPanic(t *testing.T) {
	repo := memory.NewOrderRepository()
	orderHub := hub.NewUpdateHub(4, logger.NewNop())
	uc := application.NewStreamOrderUpdates(repo, orderHub)

	now := time.Now().UTC()
	order, err := domain.NewOrder(domain.NewOrderID(), "user-1", "BTC-USDT", domain.OrderSideBuy, domain.Money{}, mustDecimal(t, "1"), now)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	if err := repo.Create(context.Background(), order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	sendErr := errors.New("send failed")
	err = uc.Execute(context.Background(), application.StreamOrderUpdatesInput{
		OrderID: order.ID,
		UserID:  "user-1",
	}, func(update application.UpdateEvent) error {
		if update.Status == domain.OrderStatusCreated {
			return sendErr
		}
		return nil
	})
	if !errors.Is(err, sendErr) {
		t.Fatalf("error = %v, want %v", err, sendErr)
	}

	// Повторная отписка через defer не должна паниковать; hub не должен держать подписчика.
	orderHub.Publish(order.ID, domain.OrderStatusFilled)
}
