package application

import (
	"context"

	"github.com/exchange-grpc/spotservice/internal/domain"
)

const (
	defaultPage     int32 = 1
	defaultPageSize int32 = 50
	maxPageSize     int32 = 100
)

// ViewMarketsInput — параметры для получения списка доступных спотовых рынков.
type ViewMarketsInput struct {
	UserRoles []string
	Page      int32
	PageSize  int32
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
func (uc *ViewMarkets) Execute(ctx context.Context, input ViewMarketsInput) ([]domain.Market, int32, error) {
	markets, err := uc.markets.ListActive(ctx)
	if err != nil {
		return nil, 0, err
	}

	filtered := make([]domain.Market, 0, len(markets))
	for _, market := range markets {
		if market.IsAccessibleBy(input.UserRoles) {
			filtered = append(filtered, market)
		}
	}

	page := input.Page
	if page <= 0 {
		page = defaultPage
	}
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	total := int32(len(filtered))
	start := (page - 1) * pageSize
	if start >= total {
		return []domain.Market{}, total, nil
	}

	end := start + pageSize
	if end > total {
		end = total
	}

	return filtered[start:end], total, nil
}
