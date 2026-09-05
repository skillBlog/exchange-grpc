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

	ctx, cancel := context.WithTimeout(integration.AuthContext(context.Background(), integration.TestUserID), 3*time.Second)
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

	ctx, cancel := context.WithTimeout(integration.AuthContext(context.Background(), integration.TestUserID), 3*time.Second)
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

	ctx, cancel := context.WithTimeout(integration.AuthContext(context.Background(), integration.TestUserID), 3*time.Second)
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

	ctx, cancel := context.WithTimeout(integration.AuthContext(context.Background(), integration.TestUserID), 3*time.Second)
	defer cancel()

	resp, err := suite.OrderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		MarketId: "BTC-USDT",
		Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
		Price:    &commonv1.Money{Amount: "100", Currency: "USD"},
		Quantity: &commonv1.Decimal{Value: "0.1"},
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	if resp.GetOrderId() == "" || resp.GetStatus() != commonv1.OrderStatus_ORDER_STATUS_CREATED {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestCreateOrder_RejectsMissingQuantity(t *testing.T) {
	suite := integration.NewSuite(t)

	ctx, cancel := context.WithTimeout(integration.AuthContext(context.Background(), integration.TestUserID), 3*time.Second)
	defer cancel()

	_, err := suite.OrderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		MarketId: "BTC-USDT",
		Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status = %v, want InvalidArgument", status.Code(err))
	}
}

func TestCreateOrder_RejectsZeroQuantity(t *testing.T) {
	suite := integration.NewSuite(t)

	ctx, cancel := context.WithTimeout(integration.AuthContext(context.Background(), integration.TestUserID), 3*time.Second)
	defer cancel()

	_, err := suite.OrderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		MarketId: "BTC-USDT",
		Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
		Quantity: &commonv1.Decimal{Value: "0"},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status = %v, want InvalidArgument", status.Code(err))
	}
}

func TestCreateOrder_RejectsZeroQuantityWithoutAuth(t *testing.T) {
	suite := integration.NewSuite(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := suite.OrderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		MarketId: "BTC-USDT",
		Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
		Quantity: &commonv1.Decimal{Value: "0"},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status = %v, want InvalidArgument (validate before jwt)", status.Code(err))
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

func TestCreateOrder_RejectsQuantityBelowMin(t *testing.T) {
	suite := integration.NewSuite(t)

	ctx, cancel := context.WithTimeout(integration.AuthContext(context.Background(), integration.TestUserID), 3*time.Second)
	defer cancel()

	_, err := suite.OrderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		MarketId: "BTC-USDT",
		Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
		Quantity: &commonv1.Decimal{Value: "0.00001"},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status = %v, want InvalidArgument", status.Code(err))
	}
}

func TestCreateOrder_RejectsQuantityPrecision(t *testing.T) {
	suite := integration.NewSuite(t)

	ctx, cancel := context.WithTimeout(integration.AuthContext(context.Background(), integration.TestUserID), 3*time.Second)
	defer cancel()

	_, err := suite.OrderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		MarketId: "BTC-USDT",
		Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
		Quantity: &commonv1.Decimal{Value: "0.000100001"},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status = %v, want InvalidArgument", status.Code(err))
	}
}

func TestCreateOrder_RejectsNotionalBelowMin(t *testing.T) {
	suite := integration.NewSuite(t)

	ctx, cancel := context.WithTimeout(integration.AuthContext(context.Background(), integration.TestUserID), 3*time.Second)
	defer cancel()

	_, err := suite.OrderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		MarketId: "BTC-USDT",
		Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
		Price:    &commonv1.Money{Amount: "100", Currency: "USD"},
		Quantity: &commonv1.Decimal{Value: "0.01"},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status = %v, want InvalidArgument", status.Code(err))
	}
}
