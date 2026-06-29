package integration_test

import (
	"context"
	"slices"
	"testing"
	"time"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	"github.com/exchange-grpc/test/integration"
)

func TestViewMarkets_ReturnsOnlyActiveMarkets(t *testing.T) {
	suite := integration.NewSuite(t, false)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := suite.SpotClient.ViewMarkets(ctx, &exchangev1.ViewMarketsRequest{
		UserRoles: []string{"trader"},
	})
	if err != nil {
		t.Fatalf("ViewMarkets() error = %v", err)
	}

	ids := make([]string, 0, len(resp.GetMarkets()))
	for _, market := range resp.GetMarkets() {
		if !market.GetEnabled() || market.GetDeletedAt() != nil {
			t.Fatalf("inactive market in response: %+v", market)
		}
		ids = append(ids, market.GetId())
	}

	want := []string{"BTC-USDT", "ETH-USDT", "BNB-USDT"}
	slices.Sort(ids)
	slices.Sort(want)
	if !slices.Equal(ids, want) {
		t.Fatalf("market ids = %v, want %v", ids, want)
	}
}
