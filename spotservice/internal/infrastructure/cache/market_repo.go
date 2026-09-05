package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/exchange-grpc/spotservice/internal/domain"
	"golang.org/x/sync/singleflight"
)

const defaultFetchTimeout = 5 * time.Second

type cacheEntry struct {
	market    domain.Market
	expiresAt time.Time
}

// MarketRepository кеширует GetByID поверх базового репозитория.
// ListActivePage не кешируется: страницы зависят от ролей пользователя.
type MarketRepository struct {
	inner        domain.MarketRepository
	ttl          time.Duration
	fetchTimeout time.Duration
	now          func() time.Time
	group        singleflight.Group

	mu   sync.RWMutex
	byID map[string]cacheEntry
}

// NewMarketRepository создаёт caching decorator.
func NewMarketRepository(inner domain.MarketRepository, ttl time.Duration) *MarketRepository {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &MarketRepository{
		inner:        inner,
		ttl:          ttl,
		fetchTimeout: defaultFetchTimeout,
		now:          time.Now,
		byID:         make(map[string]cacheEntry),
	}
}

// GetByID возвращает рынок из кеша или базового репозитория.
// Caller ctx отмена не отменяет inner-запрос (singleflight попутчики).
func (r *MarketRepository) GetByID(ctx context.Context, id string) (domain.Market, error) {
	if market, ok := r.cached(id); ok {
		return market, nil
	}

	ch := r.group.DoChan(id, func() (any, error) {
		if market, ok := r.cached(id); ok {
			return market, nil
		}

		innerCtx, cancel := context.WithTimeout(context.Background(), r.fetchTimeout)
		defer cancel()

		market, err := r.inner.GetByID(innerCtx, id)
		if err != nil {
			return domain.Market{}, err
		}

		r.mu.Lock()
		r.byID[id] = cacheEntry{market: market, expiresAt: r.now().Add(r.ttl)}
		r.mu.Unlock()
		return market, nil
	})

	select {
	case res := <-ch:
		if res.Err != nil {
			return domain.Market{}, res.Err
		}
		market, ok := res.Val.(domain.Market)
		if !ok {
			return domain.Market{}, fmt.Errorf("cache: unexpected GetByID result type %T", res.Val)
		}
		return market, nil
	case <-ctx.Done():
		return domain.Market{}, ctx.Err()
	}
}

func (r *MarketRepository) cached(id string) (domain.Market, bool) {
	r.mu.RLock()
	entry, ok := r.byID[id]
	r.mu.RUnlock()
	if !ok {
		return domain.Market{}, false
	}
	if entry.expiresAt.After(r.now()) {
		return entry.market, true
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok = r.byID[id]
	if !ok {
		return domain.Market{}, false
	}
	if !entry.expiresAt.After(r.now()) {
		delete(r.byID, id)
		return domain.Market{}, false
	}
	return entry.market, true
}

func (r *MarketRepository) cachedCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byID)
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
