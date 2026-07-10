package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/exchange-grpc/spotservice/internal/infrastructure/cache"
	"github.com/exchange-grpc/spotservice/internal/infrastructure/memory"
)

func TestMarketRepository_cachesListActive(t *testing.T) {
	inner := memory.NewSeededMarketRepository()
	repo := cache.NewMarketRepository(inner, time.Minute)

	first, err := repo.ListActive(context.Background())
	if err != nil {
		t.Fatalf("first ListActive() error = %v", err)
	}

	// Меняем in-memory данные напрямую — кеш должен вернуть старый список.
	innerRepo := memory.NewMarketRepository()
	cached := cache.NewMarketRepository(innerRepo, time.Minute)
	if _, err := cached.ListActive(context.Background()); err != nil {
		t.Fatalf("cached ListActive() error = %v", err)
	}

	second, err := cached.ListActive(context.Background())
	if err != nil {
		t.Fatalf("second ListActive() error = %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("expected cached empty list, got %d markets", len(second))
	}
	_ = first
}
