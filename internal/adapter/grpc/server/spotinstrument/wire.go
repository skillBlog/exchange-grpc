package spotinstrument

import (
	"github.com/exchange-grpc/internal/domain"
	"github.com/exchange-grpc/internal/usecase/spotinstrument"
)

// NewServerFromRepository подключает SpotInstrument handlers к репозиторию рынков.
func NewServerFromRepository(markets domain.MarketRepository) *Server {
	return NewServer(
		spotinstrument.NewViewMarkets(markets),
		spotinstrument.NewGetMarket(markets),
	)
}
