package integration_test

import (
	"context"
	"testing"
	"time"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	"github.com/exchange-grpc/internal/platform/interceptor"
	"github.com/exchange-grpc/test/integration"
)

func TestRequestID_AppearsInServerLogs(t *testing.T) {
	suite := integration.NewSuite(t, true)
	if suite.Logs == nil {
		t.Fatal("expected log observer")
	}

	const requestID = "integration-req-001"
	ctx, cancel := context.WithTimeout(
		interceptor.ContextWithRequestID(context.Background(), requestID),
		3*time.Second,
	)
	defer cancel()

	_, err := suite.SpotClient.ViewMarkets(ctx, &exchangev1.ViewMarketsRequest{})
	if err != nil {
		t.Fatalf("ViewMarkets() error = %v", err)
	}

	entries := suite.Logs.All()
	if len(entries) == 0 {
		t.Fatal("expected server log entries")
	}

	found := false
	for _, entry := range entries {
		if entry.ContextMap()["request_id"] == requestID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("request_id %q not found in logs: %+v", requestID, entries)
	}
}

func TestRequestID_PropagatesToOrderServiceLogs(t *testing.T) {
	suite := integration.NewSuite(t, true)

	const requestID = "integration-req-002"
	ctx, cancel := context.WithTimeout(
		interceptor.ContextWithRequestID(context.Background(), requestID),
		3*time.Second,
	)
	defer cancel()

	_, err := suite.OrderClient.CreateOrder(ctx, &exchangev1.CreateOrderRequest{
		UserId:    "user-1",
		MarketId:  "ETH-USDT",
		OrderType: exchangev1.OrderType_ORDER_TYPE_MARKET,
		Quantity:  "1",
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	found := false
	for _, entry := range suite.Logs.All() {
		if entry.ContextMap()["request_id"] == requestID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("request_id not found in order service logs")
	}
}
