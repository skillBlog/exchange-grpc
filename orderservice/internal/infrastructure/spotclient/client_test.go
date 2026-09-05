package spotclient

import (
	"testing"

	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
)

func TestMarketLimitsFromProto(t *testing.T) {
	got := marketLimitsFromProto(&commonv1.Market{
		MinOrderSize:      &commonv1.Decimal{Value: "0.0001"},
		QuantityPrecision: 8,
		MinNotional:       &commonv1.Decimal{Value: "10"},
	})
	if got.MinOrderSize != "0.0001" {
		t.Fatalf("MinOrderSize = %q", got.MinOrderSize)
	}
	if got.QuantityPrecision != 8 {
		t.Fatalf("QuantityPrecision = %d", got.QuantityPrecision)
	}
	if got.MinNotional != "10" {
		t.Fatalf("MinNotional = %q", got.MinNotional)
	}

	empty := marketLimitsFromProto(&commonv1.Market{})
	if empty.MinOrderSize != "" || empty.QuantityPrecision != 0 || empty.MinNotional != "" {
		t.Fatalf("empty market limits = %+v", empty)
	}
}
