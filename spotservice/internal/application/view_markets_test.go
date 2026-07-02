package application_test

import (
	"context"
	"slices"
	"testing"

	"github.com/exchange-grpc/spotservice/internal/application"
	"github.com/exchange-grpc/spotservice/internal/domain"
	"github.com/exchange-grpc/spotservice/internal/infrastructure/memory"
)

func TestViewMarkets_returnsOnlyActiveMarkets(t *testing.T) {
	repo := memory.NewSeededMarketRepository()
	uc := application.NewViewMarkets(repo)

	markets, total, err := uc.Execute(context.Background(), application.ViewMarketsInput{
		UserRoles: []string{"trader"},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	ids := marketIDs(markets)
	want := []string{"BTC-USDT", "ETH-USDT", "BNB-USDT"}
	slices.Sort(ids)
	slices.Sort(want)
	if !slices.Equal(ids, want) {
		t.Fatalf("market ids = %v, want %v", ids, want)
	}
	if total != int32(len(want)) {
		t.Fatalf("total = %d, want %d", total, len(want))
	}

	for _, market := range markets {
		if !market.IsActive() {
			t.Fatalf("market %q is not active", market.ID)
		}
	}
}

func TestViewMarkets_filtersByUserRoles(t *testing.T) {
	repo := memory.NewSeededMarketRepository()
	uc := application.NewViewMarkets(repo)

	withTrader, _, err := uc.Execute(context.Background(), application.ViewMarketsInput{
		UserRoles: []string{"trader"},
	})
	if err != nil {
		t.Fatalf("Execute(trader) error = %v", err)
	}

	withoutRoles, _, err := uc.Execute(context.Background(), application.ViewMarketsInput{})
	if err != nil {
		t.Fatalf("Execute(no roles) error = %v", err)
	}

	traderIDs := marketIDs(withTrader)
	wantTrader := []string{"BTC-USDT", "ETH-USDT", "BNB-USDT"}
	slices.Sort(traderIDs)
	slices.Sort(wantTrader)
	if !slices.Equal(traderIDs, wantTrader) {
		t.Fatalf("trader ids = %v, want %v", traderIDs, wantTrader)
	}

	noRoleIDs := marketIDs(withoutRoles)
	wantOpen := []string{"BTC-USDT", "ETH-USDT"}
	slices.Sort(noRoleIDs)
	slices.Sort(wantOpen)
	if !slices.Equal(noRoleIDs, wantOpen) {
		t.Fatalf("no-role ids = %v, want %v", noRoleIDs, wantOpen)
	}
}

func marketIDs(markets []domain.Market) []string {
	ids := make([]string, 0, len(markets))
	for _, market := range markets {
		ids = append(ids, market.ID)
	}
	slices.Sort(ids)
	return ids
}
