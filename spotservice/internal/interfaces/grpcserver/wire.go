package grpcserver

import (
	"github.com/exchange-grpc/spotservice/internal/application"
	"github.com/exchange-grpc/spotservice/internal/domain"
)

// NewServerFromRepository подключает Spot handlers к репозиторию рынков.
func NewServerFromRepository(markets domain.MarketRepository, limiter application.ViewMarketsRateLimiter) *Server {
	return NewServer(
		application.NewViewMarkets(markets, limiter),
		application.NewGetMarket(markets),
	)
}
