package spotinstrument_test

import (
	"context"
	"slices"
	"testing"

	"github.com/exchange-grpc/internal/adapter/repository/memory"
	"github.com/exchange-grpc/internal/domain"
	"github.com/exchange-grpc/internal/usecase/spotinstrument"
)

func TestViewMarkets_returnsOnlyActiveMarkets(t *testing.T) {
	repo := memory.NewSeededMarketRepository()
	uc := spotinstrument.NewViewMarkets(repo)

	markets, err := uc.Execute(context.Background(), spotinstrument.ViewMarketsInput{
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

	for _, market := range markets {
		if !market.IsActive() {
			t.Fatalf("market %q is not active", market.ID)
		}
	}
}

func TestViewMarkets_emptyWhenNoActiveMarkets(t *testing.T) {
	repo := memory.NewMarketRepository(
		domain.Market{ID: "SOL-USDT", Enabled: false},
	)
	uc := spotinstrument.NewViewMarkets(repo)

	markets, err := uc.Execute(context.Background(), spotinstrument.ViewMarketsInput{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(markets) != 0 {
		t.Fatalf("markets = %v, want empty slice", markets)
	}
}

func TestViewMarkets_filtersByUserRoles(t *testing.T) {
	repo := memory.NewSeededMarketRepository()
	uc := spotinstrument.NewViewMarkets(repo)

	withTrader, err := uc.Execute(context.Background(), spotinstrument.ViewMarketsInput{
		UserRoles: []string{"trader"},
	})
	if err != nil {
		t.Fatalf("Execute(trader) error = %v", err)
	}

	withoutRoles, err := uc.Execute(context.Background(), spotinstrument.ViewMarketsInput{})
	if err != nil {
		t.Fatalf("Execute(no roles) error = %v", err)
	}

	withAdmin, err := uc.Execute(context.Background(), spotinstrument.ViewMarketsInput{
		UserRoles: []string{"admin"},
	})
	if err != nil {
		t.Fatalf("Execute(admin) error = %v", err)
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

	if !slices.Equal(marketIDs(withAdmin), traderIDs) {
		t.Fatalf("admin ids = %v, want %v", marketIDs(withAdmin), traderIDs)
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
