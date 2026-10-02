package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/exchange-grpc/spotservice/internal/application"
	"github.com/exchange-grpc/spotservice/internal/domain"
	"github.com/exchange-grpc/spotservice/internal/infrastructure/memory"
)

func TestGetMarket_returnsOpenMarket(t *testing.T) {
	uc := application.NewGetMarket(memory.NewSeededMarketRepository(), nil)

	market, err := uc.Execute(context.Background(), application.GetMarketInput{MarketID: "BTC-USDT"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if market.ID != "BTC-USDT" {
		t.Fatalf("id = %q", market.ID)
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

func TestGetMarket_hidesRestrictedMarketAsNotFound(t *testing.T) {
	uc := application.NewGetMarket(memory.NewSeededMarketRepository(), nil)

	_, err := uc.Execute(context.Background(), application.GetMarketInput{MarketID: "BNB-USDT"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestGetMarket_allowsRestrictedMarketWithRole(t *testing.T) {
	uc := application.NewGetMarket(memory.NewSeededMarketRepository(), nil)

	market, err := uc.Execute(context.Background(), application.GetMarketInput{
		MarketID:  "BNB-USDT",
		UserRoles: []string{"trader"},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if market.ID != "BNB-USDT" {
		t.Fatalf("id = %q", market.ID)
	}
}

func TestGetMarket_wrapsNotFound(t *testing.T) {
	uc := application.NewGetMarket(memory.NewSeededMarketRepository(), nil)

	_, err := uc.Execute(context.Background(), application.GetMarketInput{MarketID: "NOPE-USDT"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
	if !strings.Contains(err.Error(), "get market") {
		t.Fatalf("error = %v, want wrap prefix", err)
	}
}

func TestGetMarket_requiresMarketID(t *testing.T) {
	uc := application.NewGetMarket(memory.NewSeededMarketRepository(), nil)

	_, err := uc.Execute(context.Background(), application.GetMarketInput{})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}
