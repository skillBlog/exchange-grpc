package cache

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/exchange-grpc/shared/roles"
	"github.com/exchange-grpc/spotservice/internal/domain"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

const (
	defaultFetchTimeout = 5 * time.Second
	defaultNegativeTTL  = 5 * time.Second
)

type cacheEntry struct {
	market    domain.Market
	err       error
	expiresAt time.Time
}

// MarketRepository кеширует GetByID поверх базового репозитория.
// Ключ GetByID включает нормализованные роли, чтобы доступ одного пользователя
// не прогрел скрытый рынок другому. ListActivePage не кешируется.
// Ошибки inner (включая NotFound) кешируются с коротким TTL, чтобы шторм запросов
// не бил в БД, пока таймаут или отсутствие рынка ещё актуально.
type MarketRepository struct {
	inner        domain.MarketRepository
	ttl          time.Duration
	negativeTTL  time.Duration
	fetchTimeout time.Duration
	now          func() time.Time
	log          *zap.Logger
	group        singleflight.Group

	mu   sync.RWMutex
	byID map[string]cacheEntry
}

// NewMarketRepository создаёт caching decorator.
func NewMarketRepository(inner domain.MarketRepository, ttl time.Duration, log *zap.Logger) *MarketRepository {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	if log == nil {
		log = zap.NewNop()
	}
	negativeTTL := defaultNegativeTTL
	if ttl < negativeTTL {
		negativeTTL = ttl
	}
	return &MarketRepository{
		inner:        inner,
		ttl:          ttl,
		negativeTTL:  negativeTTL,
		fetchTimeout: defaultFetchTimeout,
		now:          time.Now,
		log:          log,
		byID:         make(map[string]cacheEntry),
	}
}

// GetByID возвращает рынок из кеша или базового репозитория.
// Ключ кеша включает роли, чтобы доступ одного пользователя не прогрел рынок другому.
// Caller ctx отмена не отменяет inner-запрос (singleflight попутчики с тем же ключом).
// Ошибки inner, кроме NotFound, логируются внутри DoChan: отменённый caller их иначе не увидит.
func (r *MarketRepository) GetByID(ctx context.Context, id string, userRoles []string) (domain.Market, error) {
	normalizedRoles := roles.NormalizeStrings(userRoles)
	key := cacheKey(id, normalizedRoles)
	if entry, ok := r.lookup(key); ok {
		return entry.market, entry.err
	}

	ch := r.group.DoChan(key, func() (any, error) {
		if entry, ok := r.lookup(key); ok {
			return entry.market, entry.err
		}

		innerCtx, cancel := context.WithTimeout(context.Background(), r.fetchTimeout)
		defer cancel()

		market, err := r.inner.GetByID(innerCtx, id, normalizedRoles)
		if err != nil {
			r.store(key, cacheEntry{err: err, expiresAt: r.now().Add(r.negativeTTL)})
			if !errors.Is(err, domain.ErrNotFound) {
				r.log.Error("market cache get by id failed",
					zap.String("market_id", id),
					zap.Error(err),
				)
			}
			return domain.Market{}, err
		}

		r.store(key, cacheEntry{market: market, expiresAt: r.now().Add(r.ttl)})
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

func cacheKey(id string, userRoles []string) string {
	normalized := append([]string(nil), roles.NormalizeStrings(userRoles)...)
	slices.Sort(normalized)
	if len(normalized) == 0 {
		return id
	}
	return id + "\x1f" + strings.Join(normalized, ",")
}

func (r *MarketRepository) lookup(key string) (cacheEntry, bool) {
	r.mu.RLock()
	entry, ok := r.byID[key]
	r.mu.RUnlock()
	if !ok {
		return cacheEntry{}, false
	}
	if entry.expiresAt.After(r.now()) {
		return entry, true
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok = r.byID[key]
	if !ok {
		return cacheEntry{}, false
	}
	if !entry.expiresAt.After(r.now()) {
		delete(r.byID, key)
		return cacheEntry{}, false
	}
	return entry, true
}

func (r *MarketRepository) store(key string, entry cacheEntry) {
	r.mu.Lock()
	r.byID[key] = entry
	r.mu.Unlock()
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
