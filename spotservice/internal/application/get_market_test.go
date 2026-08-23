package application_test

import (
	"context"
	"errors"
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
}

func TestGetMarket_rejectsForbiddenMarket(t *testing.T) {
	uc := application.NewGetMarket(memory.NewSeededMarketRepository(), nil)

	_, err := uc.Execute(context.Background(), application.GetMarketInput{MarketID: "BNB-USDT"})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
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

func TestGetMarket_requiresMarketID(t *testing.T) {
	uc := application.NewGetMarket(memory.NewSeededMarketRepository(), nil)

	_, err := uc.Execute(context.Background(), application.GetMarketInput{})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}
