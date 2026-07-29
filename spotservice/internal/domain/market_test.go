package domain_test

import (
	"errors"
	"testing"

	"github.com/exchange-grpc/spotservice/internal/domain"
)

func TestNewMarket_normalizesRoles(t *testing.T) {
	market, err := domain.NewMarket("BTC-USDT", "Bitcoin", "BTC", "USDT", true, []string{" Trader ", "ADMIN", "Trader", "unknown"})
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

func TestNewMarket_rejectsEmptyID(t *testing.T) {
	_, err := domain.NewMarket("", "name", "BTC", "USDT", true, nil)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}
