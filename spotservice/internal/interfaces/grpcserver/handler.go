package grpcserver

import (
	"context"

	spotv1 "github.com/exchange-grpc/proto/pb/spot/v1"
	"github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/spotservice/internal/application"
)

// Server реализует spot.v1.SpotService.
type Server struct {
	spotv1.UnimplementedSpotServiceServer
	viewMarkets *application.ViewMarkets
	getMarket   *application.GetMarket
}

// NewServer создаёт gRPC-сервер Spot.
func NewServer(viewMarkets *application.ViewMarkets, getMarket *application.GetMarket) *Server {
	return &Server{
		viewMarkets: viewMarkets,
		getMarket:   getMarket,
	}
}

// ViewMarkets возвращает активные спотовые рынки, доступные вызывающей стороне.
func (s *Server) ViewMarkets(ctx context.Context, req *spotv1.ViewMarketsRequest) (*spotv1.ViewMarketsResponse, error) {
	var page, pageSize int32
	if pagination := req.GetPagination(); pagination != nil {
		page = pagination.GetPage()
		pageSize = pagination.GetPageSize()
	}

	markets, total, err := s.viewMarkets.Execute(ctx, application.ViewMarketsInput{
		UserRoles: grpc.RolesFromContext(ctx),
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &spotv1.ViewMarketsResponse{
		Markets: marketsToProto(markets),
		Total:   total,
	}, nil
}

// GetMarket возвращает рынок по идентификатору.
func (s *Server) GetMarket(ctx context.Context, req *spotv1.GetMarketRequest) (*spotv1.GetMarketResponse, error) {
	market, err := s.getMarket.Execute(ctx, req.GetMarketId())
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &spotv1.GetMarketResponse{
		Market: marketToProto(market),
	}, nil
}
