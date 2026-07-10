package application

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/exchange-grpc/spotservice/internal/domain"
)

const (
	defaultPageSize int32 = 50
	maxPageSize     int32 = 100
)

// ViewMarketsInput — параметры для получения списка доступных спотовых рынков.
type ViewMarketsInput struct {
	UserID    string
	UserRoles []string
	PageToken string
	PageSize  int32
}

// ViewMarketsOutput — результат курсорной выборки рынков.
type ViewMarketsOutput struct {
	Markets       []domain.Market
	NextPageToken string
	HasMore       bool
}

// ViewMarkets возвращает рынки, доступные для торговли в контексте пользователя.
type ViewMarkets struct {
	markets domain.MarketRepository
	limiter ViewMarketsRateLimiter
}

// NewViewMarkets создаёт use case ViewMarkets.
func NewViewMarkets(markets domain.MarketRepository, limiter ViewMarketsRateLimiter) *ViewMarkets {
	return &ViewMarkets{markets: markets, limiter: limiter}
}

// Execute возвращает активные рынки, доступные указанным ролям пользователя.
func (uc *ViewMarkets) Execute(ctx context.Context, input ViewMarketsInput) (ViewMarketsOutput, error) {
	if uc.limiter != nil && input.UserID != "" && !uc.limiter.Allow(input.UserID) {
		return ViewMarketsOutput{}, fmt.Errorf("%w: too many requests", domain.ErrRateLimited)
	}

	markets, err := uc.markets.ListActive(ctx)
	if err != nil {
		return ViewMarketsOutput{}, err
	}

	filtered := make([]domain.Market, 0, len(markets))
	for _, market := range markets {
		if market.IsAccessibleBy(input.UserRoles) {
			filtered = append(filtered, market)
		}
	}

	slices.SortFunc(filtered, func(a, b domain.Market) int {
		return strings.Compare(a.ID, b.ID)
	})

	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	start := 0
	if input.PageToken != "" {
		idx, found := slices.BinarySearchFunc(filtered, input.PageToken, func(market domain.Market, token string) int {
			return strings.Compare(market.ID, token)
		})
		if found {
			start = idx + 1
		} else if idx < len(filtered) {
			start = idx
		} else {
			return ViewMarketsOutput{Markets: []domain.Market{}}, nil
		}
	}

	end := start + int(pageSize)
	hasMore := end < len(filtered)
	if end > len(filtered) {
		end = len(filtered)
	}

	page := filtered[start:end]
	var nextPageToken string
	if hasMore && len(page) > 0 {
		nextPageToken = page[len(page)-1].ID
	}

	return ViewMarketsOutput{
		Markets:       page,
		NextPageToken: nextPageToken,
		HasMore:       hasMore,
	}, nil
}
