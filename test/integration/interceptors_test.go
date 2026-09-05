package integration_test

import (
	"context"
	"testing"
	"time"

	ordertestserver "github.com/exchange-grpc/orderservice/pkg/testserver"
	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
	orderv1 "github.com/exchange-grpc/proto/pb/order/v1"
	"github.com/exchange-grpc/test/integration"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestStreamOrderUpdates_rejectsInvalidOrderIDWithoutAuth(t *testing.T) {
	suite := integration.NewSuite(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stream, err := suite.OrderClient.StreamOrderUpdates(ctx, &orderv1.StreamOrderUpdatesRequest{
		OrderId: "not-a-uuid",
	})
	if err != nil {
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("handshake status = %v, want InvalidArgument", status.Code(err))
		}
		return
	}
	_, recvErr := stream.Recv()
	if status.Code(recvErr) != codes.InvalidArgument {
		t.Fatalf("recv status = %v, want InvalidArgument (validate before jwt)", status.Code(recvErr))
	}
}

func TestStreamOrderUpdates_receivesMultipleUpdates(t *testing.T) {
	suite := integration.NewSuite(t)

	ctx, cancel := context.WithTimeout(integration.AuthContext(context.Background(), integration.TestUserID), 5*time.Second)
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
			OrderID: created.GetOrderId(),
			UserID:  integration.TestUserID,
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

func TestHealthWatch_isPublicOnStreamInterceptor(t *testing.T) {
	suite := integration.NewSuite(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stream, err := suite.HealthClient.Watch(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	resp, err := stream.Recv()
	if err != nil {
		t.Fatalf("Watch Recv() error = %v (Watch must stay public)", err)
	}
	if resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING &&
		resp.GetStatus() != grpc_health_v1.HealthCheckResponse_NOT_SERVING &&
		resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN {
		t.Fatalf("unexpected health status %v", resp.GetStatus())
	}
}

func TestStreamUserOrderUpdates_requiresAuth(t *testing.T) {
	suite := integration.NewSuite(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stream, err := suite.OrderClient.StreamUserOrderUpdates(ctx, &orderv1.StreamUserOrderUpdatesRequest{})
	if err != nil {
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("handshake status = %v, want Unauthenticated", status.Code(err))
		}
		return
	}
	_, recvErr := stream.Recv()
	if status.Code(recvErr) != codes.Unauthenticated {
		t.Fatalf("recv status = %v, want Unauthenticated", status.Code(recvErr))
	}
}

func TestStreamUserOrderUpdates_rejectsInvalidMarketWithoutAuth(t *testing.T) {
	suite := integration.NewSuite(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stream, err := suite.OrderClient.StreamUserOrderUpdates(ctx, &orderv1.StreamUserOrderUpdatesRequest{
		MarketId: proto.String("btc-usdt"),
	})
	if err != nil {
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("handshake status = %v, want InvalidArgument", status.Code(err))
		}
		return
	}
	_, recvErr := stream.Recv()
	if status.Code(recvErr) != codes.InvalidArgument {
		t.Fatalf("recv status = %v, want InvalidArgument (validate before jwt)", status.Code(recvErr))
	}
}

func TestStreamUserOrderUpdates_receivesCreateAndFill(t *testing.T) {
	suite := integration.NewSuite(t)

	ctx, cancel := context.WithTimeout(integration.AuthContext(context.Background(), integration.TestUserID), 5*time.Second)
	defer cancel()

	stream, err := suite.OrderClient.StreamUserOrderUpdates(ctx, &orderv1.StreamUserOrderUpdatesRequest{})
	if err != nil {
		t.Fatalf("StreamUserOrderUpdates() error = %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	created, err := suite.OrderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		MarketId: "ETH-USDT",
		Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
		Quantity: &commonv1.Decimal{Value: "1"},
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	first, err := stream.Recv()
	if err != nil {
		t.Fatalf("first Recv() error = %v", err)
	}
	if first.GetOrderId() != created.GetOrderId() {
		t.Fatalf("order_id = %q, want %q", first.GetOrderId(), created.GetOrderId())
	}
	if first.GetStatus() != commonv1.OrderStatus_ORDER_STATUS_CREATED {
		t.Fatalf("first status = %v", first.GetStatus())
	}

	if err := ordertestserver.UpdateOrderStatus(context.Background(), suite.OrderServices, ordertestserver.UpdateOrderStatusInput{
		OrderID: created.GetOrderId(),
		UserID:  integration.TestUserID,
		Status:  ordertestserver.OrderStatusFilled,
	}); err != nil {
		t.Fatalf("UpdateOrderStatus() error = %v", err)
	}

	second, err := stream.Recv()
	if err != nil {
		t.Fatalf("second Recv() error = %v", err)
	}
	if second.GetStatus() != commonv1.OrderStatus_ORDER_STATUS_FILLED {
		t.Fatalf("second status = %v, want filled", second.GetStatus())
	}
}
