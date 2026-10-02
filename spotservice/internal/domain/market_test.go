package domain_test

import (
	"errors"
	"testing"

	"github.com/exchange-grpc/spotservice/internal/domain"
)

func TestNewMarket_normalizesRoles(t *testing.T) {
	market, err := domain.NewMarket("BTC-USDT", "Bitcoin", "BTC", "USDT", true, []string{" Trader ", "ADMIN", "Trader", "unknown"}, domain.Limits{})
	if err != nil {
		t.Fatalf("NewMarket() error = %v", err)
	}
	want := []string{"trader", "admin"}
	if len(market.AllowedRoles) != len(want) {
		t.Fatalf("AllowedRoles = %v, want %v", market.AllowedRoles, want)
	}
	for i := range want {
		if market.AllowedRoles[i] != want[i] {
			t.Fatalf("AllowedRoles = %v, want %v", market.AllowedRoles, want)
		}
	}
}

func TestNewMarket_setsLimits(t *testing.T) {
	market, err := domain.NewMarket("BTC-USDT", "Bitcoin", "BTC", "USDT", true, nil, domain.Limits{
		MinOrderSize:      " 0.0001 ",
		QuantityPrecision: 8,
		MinNotional:       " 10 ",
	})
	if err != nil {
		t.Fatalf("NewMarket() error = %v", err)
	}
	if market.MinOrderSize != "0.0001" {
		t.Fatalf("MinOrderSize = %q, want 0.0001", market.MinOrderSize)
	}
	if market.QuantityPrecision != 8 {
		t.Fatalf("QuantityPrecision = %d, want 8", market.QuantityPrecision)
	}
	if market.MinNotional != "10" {
		t.Fatalf("MinNotional = %q, want 10", market.MinNotional)
	}
}

func TestNewMarket_rejectsEmptyID(t *testing.T) {
	_, err := domain.NewMarket("", "name", "BTC", "USDT", true, nil, domain.Limits{})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}
