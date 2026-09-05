package application

import (
	"context"
	"fmt"
	"strings"

	sharedgrpc "github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/shared/logger"
	"github.com/exchange-grpc/shared/tracing"
	"github.com/exchange-grpc/spotservice/internal/domain"
	"go.uber.org/zap"
)

// GetMarketInput — параметры получения рынка по идентификатору.
type GetMarketInput struct {
	MarketID  string
	UserID    string
	UserRoles []string
}

// GetMarket загружает рынок по идентификатору независимо от его активности.
type GetMarket struct {
	markets domain.MarketRepository
	log     *zap.Logger
}

// NewGetMarket создаёт use case GetMarket.
func NewGetMarket(markets domain.MarketRepository, log *zap.Logger) *GetMarket {
	if log == nil {
		log = zap.NewNop()
	}
	return &GetMarket{markets: markets, log: log}
}

// Execute возвращает рынок из хранилища, если роли пользователя его допускают.
func (uc *GetMarket) Execute(ctx context.Context, input GetMarketInput) (market domain.Market, err error) {
	marketID := strings.TrimSpace(input.MarketID)
	ctx, span := tracing.Start(ctx, "spot.GetMarket",
		tracing.Attr("market.id", marketID),
		tracing.Attr("user_id", strings.TrimSpace(input.UserID)),
	)
	defer tracing.End(span, &err)

	if marketID == "" {
		return domain.Market{}, fmt.Errorf("%w: market_id is required", domain.ErrInvalidArgument)
	}

	market, err = uc.markets.GetByID(ctx, marketID)
	if err != nil {
		return domain.Market{}, err
	}
	if !market.IsAccessibleBy(input.UserRoles) {
		uc.logDenied(ctx, input.UserID, market.ID)
		return domain.Market{}, fmt.Errorf("%w: market %q", domain.ErrForbidden, market.ID)
	}

	fields := []zap.Field{
		zap.String("user_id", input.UserID),
		zap.String("market_id", market.ID),
	}
	if requestID := sharedgrpc.RequestIDFromContext(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	logger.WithTrace(ctx, uc.log).Info("get market", fields...)
	return market, nil
}

func (uc *GetMarket) logDenied(ctx context.Context, userID, marketID string) {
	fields := []zap.Field{
		zap.String("user_id", userID),
		zap.String("market_id", marketID),
		zap.String("reason", "forbidden"),
	}
	if requestID := sharedgrpc.RequestIDFromContext(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	logger.WithTrace(ctx, uc.log).Warn("get market denied", fields...)
}
