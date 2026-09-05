package grpcserver

import (
	"testing"

	"github.com/exchange-grpc/spotservice/internal/domain"
)

func TestMarketToProto_includesLimits(t *testing.T) {
	market, err := domain.NewMarket("BTC-USDT", "Bitcoin", "BTC", "USDT", true, nil)
	if err != nil {
		t.Fatalf("NewMarket() error = %v", err)
	}
	market.MinOrderSize = "0.0001"
	market.QuantityPrecision = 8
	market.MinNotional = "10"

	got := marketToProto(market)
	if got.GetMinOrderSize().GetValue() != "0.0001" {
		t.Fatalf("min_order_size = %q", got.GetMinOrderSize().GetValue())
	}
	if got.GetQuantityPrecision() != 8 {
		t.Fatalf("quantity_precision = %d", got.GetQuantityPrecision())
	}
	if got.GetMinNotional().GetValue() != "10" {
		t.Fatalf("min_notional = %q", got.GetMinNotional().GetValue())
	}
}

func TestMarketToProto_omitsEmptyLimits(t *testing.T) {
	market, err := domain.NewMarket("BTC-USDT", "Bitcoin", "BTC", "USDT", true, nil)
	if err != nil {
		t.Fatalf("NewMarket() error = %v", err)
	}

	got := marketToProto(market)
	if got.MinOrderSize != nil {
		t.Fatal("expected nil min_order_size")
	}
	if got.MinNotional != nil {
		t.Fatal("expected nil min_notional")
	}
	if got.GetQuantityPrecision() != 0 {
		t.Fatalf("quantity_precision = %d, want 0", got.GetQuantityPrecision())
	}
}
