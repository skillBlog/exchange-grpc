package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/exchange-grpc/shared/tracing"
	"github.com/exchange-grpc/spotservice/internal/domain"
)

// GetMarket загружает рынок по идентификатору независимо от его активности.
type GetMarket struct {
	markets domain.MarketRepository
}

// NewGetMarket создаёт use case GetMarket.
func NewGetMarket(markets domain.MarketRepository) *GetMarket {
	return &GetMarket{markets: markets}
}

// Execute возвращает рынок из хранилища.
func (uc *GetMarket) Execute(ctx context.Context, marketID string) (market domain.Market, err error) {
	ctx, span := tracing.Start(ctx, "spot.GetMarket", tracing.Attr("market.id", strings.TrimSpace(marketID)))
	defer tracing.End(span, &err)

	marketID = strings.TrimSpace(marketID)
	if marketID == "" {
		return domain.Market{}, fmt.Errorf("%w: market_id is required", domain.ErrInvalidArgument)
	}
	return uc.markets.GetByID(ctx, marketID)
}
