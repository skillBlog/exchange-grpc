package integration_test

import (
	"context"
	"testing"
	"time"

	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
	orderv1 "github.com/exchange-grpc/proto/pb/order/v1"
	ordertestserver "github.com/exchange-grpc/orderservice/pkg/testserver"
	"github.com/exchange-grpc/test/integration"
)

func TestStreamOrderUpdates_receivesMultipleUpdates(t *testing.T) {
	suite := integration.NewSuite(t)

	ctx, cancel := context.WithTimeout(integration.AuthContext(context.Background(), "user-1"), 5*time.Second)
	defer cancel()

	created, err := suite.OrderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		MarketId: "ETH-USDT",
		Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
		Quantity: &commonv1.Decimal{Value: "1"},
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	stream, err := suite.OrderClient.StreamOrderUpdates(ctx, &orderv1.StreamOrderUpdatesRequest{
		OrderId: created.GetOrderId(),
	})
	if err != nil {
		t.Fatalf("StreamOrderUpdates() error = %v", err)
	}

	first, err := stream.Recv()
	if err != nil {
		t.Fatalf("first Recv() error = %v", err)
	}
	if first.GetStatus() != commonv1.OrderStatus_ORDER_STATUS_CREATED {
		t.Fatalf("first status = %v", first.GetStatus())
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = ordertestserver.UpdateOrderStatus(context.Background(), suite.OrderServices, ordertestserver.UpdateOrderStatusInput{
			OrderID: created.GetOrderId().GetValue(),
			UserID:  "user-1",
			Status:  ordertestserver.OrderStatusFilled,
		})
	}()

	second, err := stream.Recv()
	if err != nil {
		t.Fatalf("second Recv() error = %v", err)
	}
	if second.GetStatus() != commonv1.OrderStatus_ORDER_STATUS_FILLED {
		t.Fatalf("second status = %v, want filled", second.GetStatus())
	}
}
