package memory

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/exchange-grpc/spotservice/internal/domain"
)

// MarketRepository хранит рынки в памяти.
type MarketRepository struct {
	mu      sync.RWMutex
	markets map[string]domain.Market
}

// NewMarketRepository создаёт репозиторий, опционально с предзагруженными рынками.
func NewMarketRepository(markets ...domain.Market) *MarketRepository {
	store := make(map[string]domain.Market, len(markets))
	for _, market := range markets {
		store[market.ID] = market
	}
	return &MarketRepository{markets: store}
}

// GetByID возвращает рынок по идентификатору независимо от активности.
func (r *MarketRepository) GetByID(_ context.Context, id string) (domain.Market, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	market, ok := r.markets[id]
	if !ok {
		return domain.Market{}, fmt.Errorf("%w: market %q", domain.ErrNotFound, id)
	}
	return market, nil
}

// ListActivePage возвращает страницу активных рынков с RBAC-фильтром и курсором по id.
func (r *MarketRepository) ListActivePage(_ context.Context, userRoles []string, limit int, afterID string) ([]domain.Market, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		return []domain.Market{}, nil
	}

	active := make([]domain.Market, 0, len(r.markets))
	for _, market := range r.markets {
		if !market.IsActive() {
			continue
		}
		if afterID != "" && strings.Compare(market.ID, afterID) <= 0 {
			continue
		}
		if !market.IsAccessibleBy(userRoles) {
			continue
		}
		active = append(active, market)
	}

	slices.SortFunc(active, func(a, b domain.Market) int {
		return strings.Compare(a.ID, b.ID)
	})

	if len(active) > limit {
		active = active[:limit]
	}
	return active, nil
}

// Ping для in-memory репозитория всегда успешен.
func (r *MarketRepository) Ping(context.Context) error {
	return nil
}

func mustMarket(id, name, base, quote string, enabled bool, allowedRoles ...string) domain.Market {
	market, err := domain.NewMarket(id, name, base, quote, enabled, allowedRoles)
	if err != nil {
		panic(err)
	}
	return market
}

// SeedMarkets возвращает набор рынков: активные и отключённые.
func SeedMarkets() []domain.Market {
	return []domain.Market{
		mustMarket("BTC-USDT", "Bitcoin / Tether", "BTC", "USDT", true),
		mustMarket("ETH-USDT", "Ethereum / Tether", "ETH", "USDT", true),
		mustMarket("BNB-USDT", "BNB / Tether", "BNB", "USDT", true, "trader", "admin"),
		mustMarket("SOL-USDT", "Solana / Tether", "SOL", "USDT", false),
		mustMarket("XRP-USDT", "Ripple / Tether", "XRP", "USDT", false),
	}
}

// NewSeededMarketRepository возвращает репозиторий с предзагруженными SeedMarkets.
func NewSeededMarketRepository() *MarketRepository {
	return NewMarketRepository(SeedMarkets()...)
}

var _ domain.MarketRepository = (*MarketRepository)(nil)
