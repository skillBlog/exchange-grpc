package grpcserver

import (
	"github.com/exchange-grpc/spotservice/internal/application"
	"github.com/exchange-grpc/spotservice/internal/domain"
	"go.uber.org/zap"
)

// NewServerFromRepository подключает Spot handlers к репозиторию рынков.
func NewServerFromRepository(markets domain.MarketRepository, limiter application.ViewMarketsRateLimiter, log *zap.Logger) *Server {
	return NewServer(
		application.NewViewMarkets(markets, limiter, log),
		application.NewGetMarket(markets, log),
	)
}
