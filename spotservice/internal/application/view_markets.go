package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/exchange-grpc/shared/roles"
	"github.com/exchange-grpc/shared/tracing"
	"github.com/exchange-grpc/spotservice/internal/domain"
	"go.uber.org/zap"
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
	log     *zap.Logger
}

// NewViewMarkets создаёт use case ViewMarkets.
func NewViewMarkets(markets domain.MarketRepository, limiter ViewMarketsRateLimiter, log *zap.Logger) *ViewMarkets {
	if log == nil {
		log = zap.NewNop()
	}
	return &ViewMarkets{markets: markets, limiter: limiter, log: log}
}

// Execute возвращает страницу активных рынков с фильтрацией и пагинацией на уровне репозитория.
func (uc *ViewMarkets) Execute(ctx context.Context, input ViewMarketsInput) (out ViewMarketsOutput, err error) {
	ctx, span := tracing.Start(ctx, "spot.ViewMarkets",
		tracing.Attr("user_id", strings.TrimSpace(input.UserID)),
	)
	defer tracing.End(span, &err)

	if err = checkViewMarketsRateLimit(ctx, uc.limiter, input.UserID); err != nil {
		return ViewMarketsOutput{}, fmt.Errorf("view markets rate limit: %w", err)
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
		return ViewMarketsOutput{}, fmt.Errorf("list markets: %w", err)
	}

	hasMore := len(markets) > int(pageSize)
	if hasMore {
		markets = markets[:pageSize]
	}

	var nextPageToken string
	if hasMore && len(markets) > 0 {
		nextPageToken = markets[len(markets)-1].ID
	}

	logAudit(ctx, uc.log, "view markets", input.UserID, zap.Int("count", len(markets)))

	return ViewMarketsOutput{
		Markets:       markets,
		NextPageToken: nextPageToken,
		HasMore:       hasMore,
	}, nil
}
