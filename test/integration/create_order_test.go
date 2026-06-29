package integration_test

import (
	"context"
	"testing"
	"time"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	"github.com/exchange-grpc/test/integration"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateOrder_RejectsInactiveMarket(t *testing.T) {
	suite := integration.NewSuite(t, false)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := suite.OrderClient.CreateOrder(ctx, &exchangev1.CreateOrderRequest{
		UserId:    "user-1",
		MarketId:  "SOL-USDT",
		OrderType: exchangev1.OrderType_ORDER_TYPE_MARKET,
		Quantity:  "1",
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status = %v, want FailedPrecondition", status.Code(err))
	}
}

func TestCreateOrder_RejectsDeletedMarket(t *testing.T) {
	suite := integration.NewSuite(t, false)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := suite.OrderClient.CreateOrder(ctx, &exchangev1.CreateOrderRequest{
		UserId:    "user-1",
		MarketId:  "XRP-USDT",
		OrderType: exchangev1.OrderType_ORDER_TYPE_MARKET,
		Quantity:  "1",
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status = %v, want FailedPrecondition", status.Code(err))
	}
}

func TestCreateOrder_RejectsUnknownMarket(t *testing.T) {
	suite := integration.NewSuite(t, false)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := suite.OrderClient.CreateOrder(ctx, &exchangev1.CreateOrderRequest{
		UserId:    "user-1",
		MarketId:  "DOGE-USDT",
		OrderType: exchangev1.OrderType_ORDER_TYPE_MARKET,
		Quantity:  "1",
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("status = %v, want NotFound", status.Code(err))
	}
}

func TestCreateOrder_SucceedsForActiveMarket(t *testing.T) {
	suite := integration.NewSuite(t, false)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := suite.OrderClient.CreateOrder(ctx, &exchangev1.CreateOrderRequest{
		UserId:    "user-1",
		MarketId:  "BTC-USDT",
		OrderType: exchangev1.OrderType_ORDER_TYPE_LIMIT,
		Price:     "100",
		Quantity:  "0.01",
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	if resp.GetOrderId() == "" || resp.GetStatus() != "created" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}
