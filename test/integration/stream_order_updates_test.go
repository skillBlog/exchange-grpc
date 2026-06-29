package integration_test

import (
	"context"
	"testing"
	"time"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	"github.com/exchange-grpc/internal/domain"
	orderuc "github.com/exchange-grpc/internal/usecase/order"
	"github.com/exchange-grpc/test/integration"
)

func TestStreamOrderUpdates_receivesMultipleUpdates(t *testing.T) {
	suite := integration.NewSuite(t, false)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	created, err := suite.OrderClient.CreateOrder(ctx, &exchangev1.CreateOrderRequest{
		UserId:    "user-1",
		MarketId:  "ETH-USDT",
		OrderType: exchangev1.OrderType_ORDER_TYPE_MARKET,
		Quantity:  "1",
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	stream, err := suite.OrderClient.StreamOrderUpdates(ctx, &exchangev1.StreamOrderUpdatesRequest{
		OrderId: created.GetOrderId(),
		UserId:  "user-1",
	})
	if err != nil {
		t.Fatalf("StreamOrderUpdates() error = %v", err)
	}

	first, err := stream.Recv()
	if err != nil {
		t.Fatalf("first Recv() error = %v", err)
	}
	if first.GetStatus() != string(domain.OrderStatusCreated) {
		t.Fatalf("first status = %q", first.GetStatus())
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = suite.OrderServices.UpdateOrderStatus.Execute(context.Background(), orderuc.UpdateOrderStatusInput{
			OrderID: created.GetOrderId(),
			UserID:  "user-1",
			Status:  domain.OrderStatusFilled,
		})
	}()

	second, err := stream.Recv()
	if err != nil {
		t.Fatalf("second Recv() error = %v", err)
	}
	if second.GetStatus() != string(domain.OrderStatusFilled) {
		t.Fatalf("second status = %q, want filled", second.GetStatus())
	}
}
