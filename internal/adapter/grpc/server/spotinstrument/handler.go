package spotinstrument

import (
	"context"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	"github.com/exchange-grpc/internal/adapter/grpc/mapper"
	"github.com/exchange-grpc/internal/usecase/spotinstrument"
)

// Server реализует exchange.v1.SpotInstrumentService.
type Server struct {
	exchangev1.UnimplementedSpotInstrumentServiceServer
	viewMarkets *spotinstrument.ViewMarkets
	getMarket   *spotinstrument.GetMarket
}

// NewServer создаёт gRPC-сервер SpotInstrument.
func NewServer(viewMarkets *spotinstrument.ViewMarkets, getMarket *spotinstrument.GetMarket) *Server {
	return &Server{
		viewMarkets: viewMarkets,
		getMarket:   getMarket,
	}
}

// ViewMarkets возвращает активные спотовые рынки, доступные вызывающей стороне.
func (s *Server) ViewMarkets(ctx context.Context, req *exchangev1.ViewMarketsRequest) (*exchangev1.ViewMarketsResponse, error) {
	markets, err := s.viewMarkets.Execute(ctx, spotinstrument.ViewMarketsInput{
		UserRoles: req.GetUserRoles(),
	})
	if err != nil {
		return nil, mapper.ToGRPCError(err)
	}

	return &exchangev1.ViewMarketsResponse{
		Markets: mapper.MarketsToProto(markets),
	}, nil
}

// GetMarket возвращает рынок по идентификатору.
func (s *Server) GetMarket(ctx context.Context, req *exchangev1.GetMarketRequest) (*exchangev1.GetMarketResponse, error) {
	market, err := s.getMarket.Execute(ctx, req.GetMarketId())
	if err != nil {
		return nil, mapper.ToGRPCError(err)
	}

	return &exchangev1.GetMarketResponse{
		Market: mapper.MarketToProto(market),
	}, nil
}
