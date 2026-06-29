package integration_test

import (
	"context"
	"slices"
	"testing"
	"time"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	"github.com/exchange-grpc/test/integration"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestViewMarkets_FiltersByUserRoles(t *testing.T) {
	suite := integration.NewSuite(t, false)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := suite.SpotClient.ViewMarkets(ctx, &exchangev1.ViewMarketsRequest{})
	if err != nil {
		t.Fatalf("ViewMarkets() error = %v", err)
	}

	ids := make([]string, 0, len(resp.GetMarkets()))
	for _, market := range resp.GetMarkets() {
		ids = append(ids, market.GetId())
	}

	want := []string{"BTC-USDT", "ETH-USDT"}
	slices.Sort(ids)
	slices.Sort(want)
	if !slices.Equal(ids, want) {
		t.Fatalf("market ids = %v, want %v", ids, want)
	}
}

func TestCreateOrder_RejectsForbiddenMarket(t *testing.T) {
	suite := integration.NewSuite(t, false)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := suite.OrderClient.CreateOrder(ctx, &exchangev1.CreateOrderRequest{
		UserId:    "user-1",
		MarketId:  "BNB-USDT",
		OrderType: exchangev1.OrderType_ORDER_TYPE_MARKET,
		Quantity:  "1",
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status = %v, want PermissionDenied", status.Code(err))
	}

	resp, err := suite.OrderClient.CreateOrder(ctx, &exchangev1.CreateOrderRequest{
		UserId:    "user-1",
		MarketId:  "BNB-USDT",
		OrderType: exchangev1.OrderType_ORDER_TYPE_MARKET,
		Quantity:  "1",
		UserRoles: []string{"trader"},
	})
	if err != nil {
		t.Fatalf("CreateOrder(trader) error = %v", err)
	}
	if resp.GetOrderId() == "" || resp.GetStatus() != "created" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}
