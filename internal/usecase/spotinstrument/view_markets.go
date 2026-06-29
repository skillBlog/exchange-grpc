package spotinstrument

import (
	"context"

	"github.com/exchange-grpc/internal/domain"
)

// ViewMarketsInput — параметры для получения списка доступных спотовых рынков.
type ViewMarketsInput struct {
	UserRoles []string
}

// ViewMarkets возвращает рынки, доступные для торговли в контексте пользователя.
type ViewMarkets struct {
	markets domain.MarketRepository
}

// NewViewMarkets создаёт use case ViewMarkets.
func NewViewMarkets(markets domain.MarketRepository) *ViewMarkets {
	return &ViewMarkets{markets: markets}
}

// Execute возвращает активные рынки, доступные указанным ролям пользователя.
func (uc *ViewMarkets) Execute(ctx context.Context, input ViewMarketsInput) ([]domain.Market, error) {
	markets, err := uc.markets.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]domain.Market, 0, len(markets))
	for _, market := range markets {
		if market.IsAccessibleBy(input.UserRoles) {
			filtered = append(filtered, market)
		}
	}

	return filtered, nil
}
