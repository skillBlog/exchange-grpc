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
	uc := application.NewViewMarkets(repo, nil)

	out, err := uc.Execute(context.Background(), application.ViewMarketsInput{
		UserRoles: []string{"trader"},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	ids := marketIDs(out.Markets)
	want := []string{"BTC-USDT", "ETH-USDT", "BNB-USDT"}
	slices.Sort(ids)
	slices.Sort(want)
	if !slices.Equal(ids, want) {
		t.Fatalf("market ids = %v, want %v", ids, want)
	}

	for _, market := range out.Markets {
		if !market.IsActive() {
			t.Fatalf("market %q is not active", market.ID)
		}
	}
}

func TestViewMarkets_filtersByUserRoles(t *testing.T) {
	repo := memory.NewSeededMarketRepository()
	uc := application.NewViewMarkets(repo, nil)

	withTrader, err := uc.Execute(context.Background(), application.ViewMarketsInput{
		UserRoles: []string{"trader"},
	})
	if err != nil {
		t.Fatalf("Execute(trader) error = %v", err)
	}

	withoutRoles, err := uc.Execute(context.Background(), application.ViewMarketsInput{})
	if err != nil {
		t.Fatalf("Execute(no roles) error = %v", err)
	}

	traderIDs := marketIDs(withTrader.Markets)
	wantTrader := []string{"BTC-USDT", "ETH-USDT", "BNB-USDT"}
	slices.Sort(traderIDs)
	slices.Sort(wantTrader)
	if !slices.Equal(traderIDs, wantTrader) {
		t.Fatalf("trader ids = %v, want %v", traderIDs, wantTrader)
	}

	noRoleIDs := marketIDs(withoutRoles.Markets)
	wantOpen := []string{"BTC-USDT", "ETH-USDT"}
	slices.Sort(noRoleIDs)
	slices.Sort(wantOpen)
	if !slices.Equal(noRoleIDs, wantOpen) {
		t.Fatalf("no-role ids = %v, want %v", noRoleIDs, wantOpen)
	}
}

func TestViewMarkets_cursorPagination(t *testing.T) {
	repo := memory.NewSeededMarketRepository()
	uc := application.NewViewMarkets(repo, nil)

	first, err := uc.Execute(context.Background(), application.ViewMarketsInput{
		UserRoles: []string{"trader"},
		PageSize:  2,
	})
	if err != nil {
		t.Fatalf("first page error = %v", err)
	}
	if !first.HasMore || first.NextPageToken == "" {
		t.Fatalf("expected more pages: %+v", first)
	}
	if len(first.Markets) != 2 {
		t.Fatalf("first page size = %d, want 2", len(first.Markets))
	}

	second, err := uc.Execute(context.Background(), application.ViewMarketsInput{
		UserRoles: []string{"trader"},
		PageToken: first.NextPageToken,
		PageSize:  2,
	})
	if err != nil {
		t.Fatalf("second page error = %v", err)
	}
	if second.HasMore {
		t.Fatalf("expected last page, got %+v", second)
	}
	if len(second.Markets) != 1 {
		t.Fatalf("second page size = %d, want 1", len(second.Markets))
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
