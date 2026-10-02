package grpcserver

import (
	"testing"

	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
	"github.com/exchange-grpc/spotservice/internal/domain"
)

func TestMarketToProto_includesLimits(t *testing.T) {
	market, err := domain.NewMarket("BTC-USDT", "Bitcoin", "BTC", "USDT", true, nil, domain.Limits{
		MinOrderSize:      "0.0001",
		QuantityPrecision: 8,
		MinNotional:       "10",
	})
	if err != nil {
		t.Fatalf("NewMarket() error = %v", err)
	}

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
	market, err := domain.NewMarket("BTC-USDT", "Bitcoin", "BTC", "USDT", true, nil, domain.Limits{})
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

func TestRolesToProto_unknownBecomesUnspecified(t *testing.T) {
	got := rolesToProto([]string{"user", "auditor", "trader", ""})
	want := []commonv1.Role{
		commonv1.Role_ROLE_USER,
		commonv1.Role_ROLE_UNSPECIFIED,
		commonv1.Role_ROLE_TRADER,
	}
	if len(got) != len(want) {
		t.Fatalf("roles = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("roles = %v, want %v", got, want)
		}
	}
}
