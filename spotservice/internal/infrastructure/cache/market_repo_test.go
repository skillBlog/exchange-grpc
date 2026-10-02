package cache_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/exchange-grpc/spotservice/internal/domain"
	"github.com/exchange-grpc/spotservice/internal/infrastructure/cache"
	"github.com/exchange-grpc/spotservice/internal/infrastructure/memory"
)

func TestMarketRepository_cachesGetByID(t *testing.T) {
	inner := memory.NewSeededMarketRepository()
	repo := cache.NewMarketRepository(inner, time.Minute, nil)

	first, err := repo.GetByID(context.Background(), "BTC-USDT", nil)
	if err != nil {
		t.Fatalf("first GetByID() error = %v", err)
	}
	if first.ID != "BTC-USDT" {
		t.Fatalf("id = %q", first.ID)
	}

	second, err := repo.GetByID(context.Background(), "BTC-USDT", nil)
	if err != nil {
		t.Fatalf("second GetByID() error = %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("cached id mismatch: %q vs %q", second.ID, first.ID)
	}
}

func TestMarketRepository_listActivePagePassesThrough(t *testing.T) {
	inner := memory.NewSeededMarketRepository()
	repo := cache.NewMarketRepository(inner, time.Minute, nil)

	page, err := repo.ListActivePage(context.Background(), []string{"trader"}, 2, "")
	if err != nil {
		t.Fatalf("ListActivePage() error = %v", err)
	}
	if len(page) != 2 {
		t.Fatalf("page size = %d, want 2", len(page))
	}
}

func TestMarketRepository_pingDelegates(t *testing.T) {
	repo := cache.NewMarketRepository(memory.NewSeededMarketRepository(), time.Minute, nil)
	if err := repo.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func TestMarketRepository_isolatesRestrictedMarketByRoles(t *testing.T) {
	repo := cache.NewMarketRepository(memory.NewSeededMarketRepository(), time.Minute, nil)

	trader, err := repo.GetByID(context.Background(), "BNB-USDT", []string{"trader"})
	if err != nil {
		t.Fatalf("trader GetByID() error = %v", err)
	}
	if trader.ID != "BNB-USDT" {
		t.Fatalf("id = %q", trader.ID)
	}

	_, err = repo.GetByID(context.Background(), "BNB-USDT", nil)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("user GetByID() error = %v, want ErrNotFound", err)
	}

	_, err = repo.GetByID(context.Background(), "BNB-USDT", []string{"user"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("role user GetByID() error = %v, want ErrNotFound", err)
	}
}
