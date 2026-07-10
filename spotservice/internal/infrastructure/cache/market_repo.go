package cache

import (
	"context"
	"sync"
	"time"

	"github.com/exchange-grpc/spotservice/internal/domain"
)

type cacheEntry struct {
	market    domain.Market
	expiresAt time.Time
}

// MarketRepository кеширует чтение рынков поверх базового репозитория.
type MarketRepository struct {
	inner domain.MarketRepository
	ttl   time.Duration
	now   func() time.Time

	mu            sync.RWMutex
	activeList    []domain.Market
	activeExpires time.Time
	byID          map[string]cacheEntry
}

// NewMarketRepository создаёт caching decorator.
func NewMarketRepository(inner domain.MarketRepository, ttl time.Duration) *MarketRepository {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &MarketRepository{
		inner: inner,
		ttl:   ttl,
		now:   time.Now,
		byID:  make(map[string]cacheEntry),
	}
}

// GetByID возвращает рынок из кеша или базового репозитория.
func (r *MarketRepository) GetByID(ctx context.Context, id string) (domain.Market, error) {
	now := r.now()

	r.mu.RLock()
	if entry, ok := r.byID[id]; ok && entry.expiresAt.After(now) {
		r.mu.RUnlock()
		return entry.market, nil
	}
	r.mu.RUnlock()

	market, err := r.inner.GetByID(ctx, id)
	if err != nil {
		return domain.Market{}, err
	}

	r.mu.Lock()
	r.byID[id] = cacheEntry{market: market, expiresAt: now.Add(r.ttl)}
	r.mu.Unlock()
	return market, nil
}

// ListActive возвращает активные рынки из кеша или базового репозитория.
func (r *MarketRepository) ListActive(ctx context.Context) ([]domain.Market, error) {
	now := r.now()

	r.mu.RLock()
	if r.activeExpires.After(now) && r.activeList != nil {
		cached := append([]domain.Market(nil), r.activeList...)
		r.mu.RUnlock()
		return cached, nil
	}
	r.mu.RUnlock()

	markets, err := r.inner.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.activeList = append([]domain.Market(nil), markets...)
	r.activeExpires = now.Add(r.ttl)
	for _, market := range markets {
		r.byID[market.ID] = cacheEntry{market: market, expiresAt: r.activeExpires}
	}
	r.mu.Unlock()
	return markets, nil
}

// Ping проверяет, что кеш и базовый репозиторий отвечают.
func (r *MarketRepository) Ping(ctx context.Context) error {
	_, err := r.ListActive(ctx)
	return err
}

var _ domain.MarketRepository = (*MarketRepository)(nil)
