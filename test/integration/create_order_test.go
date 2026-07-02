package integration_test

import (
	"context"
	"testing"
	"time"

	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
	orderv1 "github.com/exchange-grpc/proto/pb/order/v1"
	"github.com/exchange-grpc/test/integration"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateOrder_RejectsInactiveMarket(t *testing.T) {
	suite := integration.NewSuite(t)

	ctx, cancel := context.WithTimeout(integration.AuthContext(context.Background(), "user-1"), 3*time.Second)
	defer cancel()

	_, err := suite.OrderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		MarketId: "SOL-USDT",
		Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
		Quantity: &commonv1.Decimal{Value: "1"},
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status = %v, want FailedPrecondition", status.Code(err))
	}
}

func TestCreateOrder_RejectsDeletedMarket(t *testing.T) {
	suite := integration.NewSuite(t)

	ctx, cancel := context.WithTimeout(integration.AuthContext(context.Background(), "user-1"), 3*time.Second)
	defer cancel()

	_, err := suite.OrderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		MarketId: "XRP-USDT",
		Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
		Quantity: &commonv1.Decimal{Value: "1"},
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status = %v, want FailedPrecondition", status.Code(err))
	}
}

func TestCreateOrder_RejectsUnknownMarket(t *testing.T) {
	suite := integration.NewSuite(t)

	ctx, cancel := context.WithTimeout(integration.AuthContext(context.Background(), "user-1"), 3*time.Second)
	defer cancel()

	_, err := suite.OrderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		MarketId: "DOGE-USDT",
		Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
		Quantity: &commonv1.Decimal{Value: "1"},
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("status = %v, want NotFound", status.Code(err))
	}
}

func TestCreateOrder_SucceedsForActiveMarket(t *testing.T) {
	suite := integration.NewSuite(t)

	ctx, cancel := context.WithTimeout(integration.AuthContext(context.Background(), "user-1"), 3*time.Second)
	defer cancel()

	resp, err := suite.OrderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		MarketId: "BTC-USDT",
		Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
		Price:    &commonv1.Money{Amount: "100", Currency: "USD"},
		Quantity: &commonv1.Decimal{Value: "0.01"},
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	if resp.GetOrderId().GetValue() == "" || resp.GetStatus() != commonv1.OrderStatus_ORDER_STATUS_CREATED {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestCreateOrder_RequiresAuth(t *testing.T) {
	suite := integration.NewSuite(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := suite.OrderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		MarketId: "BTC-USDT",
		Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
		Quantity: &commonv1.Decimal{Value: "0.01"},
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("status = %v, want Unauthenticated", status.Code(err))
	}
}
