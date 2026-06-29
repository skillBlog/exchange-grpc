package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/exchange-grpc/internal/domain"
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

// ListActive возвращает включённые и не помеченные как удалённые рынки.
func (r *MarketRepository) ListActive(_ context.Context) ([]domain.Market, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	active := make([]domain.Market, 0, len(r.markets))
	for _, market := range r.markets {
		if market.IsActive() {
			active = append(active, market)
		}
	}
	return active, nil
}

// Upsert вставляет или заменяет рынок. Предназначен для тестов и seed-данных.
func (r *MarketRepository) Upsert(market domain.Market) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.markets[market.ID] = market
}

// referenceSeedTime фиксирует deleted_at для стабильности тестов между перезапусками.
var referenceSeedTime = time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

// SeedMarkets возвращает набор рынков: активные, отключённые и удалённые.
func SeedMarkets() []domain.Market {
	deletedAt := referenceSeedTime
	return []domain.Market{
		{
			ID:         "BTC-USDT",
			Name:       "Bitcoin / Tether",
			BaseAsset:  "BTC",
			QuoteAsset: "USDT",
			Enabled:    true,
		},
		{
			ID:         "ETH-USDT",
			Name:       "Ethereum / Tether",
			BaseAsset:  "ETH",
			QuoteAsset: "USDT",
			Enabled:    true,
		},
		{
			ID:           "BNB-USDT",
			Name:         "BNB / Tether",
			BaseAsset:    "BNB",
			QuoteAsset:   "USDT",
			Enabled:      true,
			AllowedRoles: []string{"trader", "admin"},
		},
		{
			ID:         "SOL-USDT",
			Name:       "Solana / Tether",
			BaseAsset:  "SOL",
			QuoteAsset: "USDT",
			Enabled:    false,
		},
		{
			ID:         "XRP-USDT",
			Name:       "Ripple / Tether",
			BaseAsset:  "XRP",
			QuoteAsset: "USDT",
			Enabled:    true,
			DeletedAt:  &deletedAt,
		},
	}
}

// NewSeededMarketRepository возвращает репозиторий с предзагруженными SeedMarkets.
func NewSeededMarketRepository() *MarketRepository {
	return NewMarketRepository(SeedMarkets()...)
}

var _ domain.MarketRepository = (*MarketRepository)(nil)
