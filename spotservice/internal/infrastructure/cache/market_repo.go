package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/exchange-grpc/spotservice/internal/domain"
	"golang.org/x/sync/singleflight"
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
	group singleflight.Group

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
// Lock не удерживается на время обращения к БД; TTL считается от успешного чтения.
func (r *MarketRepository) GetByID(ctx context.Context, id string) (domain.Market, error) {
	if market, ok := r.cached(id); ok {
		return market, nil
	}

	value, err, _ := r.group.Do(id, func() (any, error) {
		if market, ok := r.cached(id); ok {
			return market, nil
		}

		market, err := r.inner.GetByID(ctx, id)
		if err != nil {
			return domain.Market{}, err
		}

		r.mu.Lock()
		r.byID[id] = cacheEntry{market: market, expiresAt: r.now().Add(r.ttl)}
		r.mu.Unlock()
		return market, nil
	})
	if err != nil {
		return domain.Market{}, err
	}
	market, ok := value.(domain.Market)
	if !ok {
		return domain.Market{}, fmt.Errorf("cache: unexpected GetByID result type %T", value)
	}
	return market, nil
}

func (r *MarketRepository) cached(id string) (domain.Market, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.byID[id]
	if !ok || !entry.expiresAt.After(r.now()) {
		return domain.Market{}, false
	}
	return entry.market, true
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
