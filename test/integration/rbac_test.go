package integration_test

import (
	"context"
	"slices"
	"testing"
	"time"

	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
	orderv1 "github.com/exchange-grpc/proto/pb/order/v1"
	spotv1 "github.com/exchange-grpc/proto/pb/spot/v1"
	"github.com/exchange-grpc/test/integration"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestViewMarkets_FiltersByUserRoles(t *testing.T) {
	suite := integration.NewSuite(t)

	ctx, cancel := context.WithTimeout(integration.AuthContext(context.Background(), "user-1"), 3*time.Second)
	defer cancel()

	resp, err := suite.SpotClient.ViewMarkets(ctx, &spotv1.ViewMarketsRequest{})
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
	suite := integration.NewSuite(t)

	ctx, cancel := context.WithTimeout(integration.AuthContext(context.Background(), "user-1"), 3*time.Second)
	defer cancel()

	_, err := suite.OrderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		MarketId: "BNB-USDT",
		Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
		Quantity: &commonv1.Decimal{Value: "1"},
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status = %v, want PermissionDenied", status.Code(err))
	}

	ctxTrader, cancelTrader := context.WithTimeout(integration.AuthContext(context.Background(), "user-1", "trader"), 3*time.Second)
	defer cancelTrader()

	resp, err := suite.OrderClient.CreateOrder(ctxTrader, &orderv1.CreateOrderRequest{
		MarketId: "BNB-USDT",
		Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
		Quantity: &commonv1.Decimal{Value: "1"},
	})
	if err != nil {
		t.Fatalf("CreateOrder(trader) error = %v", err)
	}
	if resp.GetOrderId().GetValue() == "" || resp.GetStatus() != commonv1.OrderStatus_ORDER_STATUS_CREATED {
		t.Fatalf("unexpected response: %+v", resp)
	}
}
