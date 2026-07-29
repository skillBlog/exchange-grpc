package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/exchange-grpc/shared/roles"
	"github.com/exchange-grpc/shared/tracing"
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

// Execute возвращает страницу активных рынков с фильтрацией и пагинацией на уровне репозитория.
func (uc *ViewMarkets) Execute(ctx context.Context, input ViewMarketsInput) (out ViewMarketsOutput, err error) {
	ctx, span := tracing.Start(ctx, "spot.ViewMarkets")
	defer tracing.End(span, &err)

	if uc.limiter != nil && input.UserID != "" && !uc.limiter.Allow(input.UserID) {
		return ViewMarketsOutput{}, fmt.Errorf("%w: too many requests", domain.ErrRateLimited)
	}

	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	userRoles := roles.NormalizeStrings(input.UserRoles)
	markets, err := uc.markets.ListActivePage(ctx, userRoles, int(pageSize)+1, strings.TrimSpace(input.PageToken))
	if err != nil {
		return ViewMarketsOutput{}, err
	}

	hasMore := len(markets) > int(pageSize)
	if hasMore {
		markets = markets[:pageSize]
	}

	var nextPageToken string
	if hasMore && len(markets) > 0 {
		nextPageToken = markets[len(markets)-1].ID
	}

	return ViewMarketsOutput{
		Markets:       markets,
		NextPageToken: nextPageToken,
		HasMore:       hasMore,
	}, nil
}
