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

// MarketRepository кеширует GetByID поверх базового репозитория.
// ListActivePage не кешируется: страницы зависят от ролей пользователя.
type MarketRepository struct {
	inner domain.MarketRepository
	ttl   time.Duration
	now   func() time.Time

	mu   sync.RWMutex
	byID map[string]cacheEntry
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

// ListActivePage проксирует запрос в базовый репозиторий без кеша списка.
func (r *MarketRepository) ListActivePage(ctx context.Context, userRoles []string, limit int, afterID string) ([]domain.Market, error) {
	return r.inner.ListActivePage(ctx, userRoles, limit, afterID)
}

// Ping делегирует проверку базовому репозиторию.
func (r *MarketRepository) Ping(ctx context.Context) error {
	return r.inner.Ping(ctx)
}

var _ domain.MarketRepository = (*MarketRepository)(nil)
