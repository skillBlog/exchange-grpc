package integration_test

import (
	"context"
	"testing"
	"time"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	"github.com/exchange-grpc/test/integration"
)

func TestViewMarkets_IncrementsPrometheusCounter(t *testing.T) {
	suite := integration.NewSuite(t, false)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if _, err := suite.SpotClient.ViewMarkets(ctx, &exchangev1.ViewMarketsRequest{}); err != nil {
		t.Fatalf("ViewMarkets() error = %v", err)
	}

	body := integration.ScrapeMetrics(t)
	if !integration.ContainsMetric(body, "grpc_server_handled_total", "ViewMarkets") {
		t.Fatalf("metrics missing ViewMarkets counter:\n%s", body)
	}
}

func TestCreateOrder_IncrementsPrometheusCounter(t *testing.T) {
	suite := integration.NewSuite(t, false)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := suite.OrderClient.CreateOrder(ctx, &exchangev1.CreateOrderRequest{
		UserId:    "user-1",
		MarketId:  "BTC-USDT",
		OrderType: exchangev1.OrderType_ORDER_TYPE_MARKET,
		Quantity:  "0.5",
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	body := integration.ScrapeMetrics(t)
	if !integration.ContainsMetric(body, "grpc_server_handled_total", "CreateOrder") {
		t.Fatalf("metrics missing CreateOrder counter:\n%s", body)
	}
}
