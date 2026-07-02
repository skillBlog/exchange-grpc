package application

import (
	"context"

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
func (uc *GetMarket) Execute(ctx context.Context, marketID string) (domain.Market, error) {
	return uc.markets.GetByID(ctx, marketID)
}
