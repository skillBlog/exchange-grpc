package order

import (
	"context"
	"errors"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	"github.com/exchange-grpc/internal/adapter/grpc/mapper"
	"github.com/exchange-grpc/internal/domain"
	orderuc "github.com/exchange-grpc/internal/usecase/order"
)

// Server реализует exchange.v1.OrderService.
type Server struct {
	exchangev1.UnimplementedOrderServiceServer
	createOrder        *orderuc.CreateOrder
	getOrderStatus     *orderuc.GetOrderStatus
	streamOrderUpdates *orderuc.StreamOrderUpdates
}

// NewServer создаёт gRPC-сервер Order.
func NewServer(services Services) *Server {
	return &Server{
		createOrder:        services.CreateOrder,
		getOrderStatus:     services.GetOrderStatus,
		streamOrderUpdates: services.StreamOrderUpdates,
	}
}

// CreateOrder создаёт ордер после проверки целевого рынка.
func (s *Server) CreateOrder(ctx context.Context, req *exchangev1.CreateOrderRequest) (*exchangev1.CreateOrderResponse, error) {
	orderType, err := mapper.OrderTypeToDomain(req.GetOrderType())
	if err != nil {
		return nil, mapper.ToGRPCError(err)
	}

	out, err := s.createOrder.Execute(ctx, orderuc.CreateOrderInput{
		UserID:    req.GetUserId(),
		MarketID:  req.GetMarketId(),
		Type:      orderType,
		Price:     req.GetPrice(),
		Quantity:  req.GetQuantity(),
		UserRoles: req.GetUserRoles(),
	})
	if err != nil {
		return nil, mapper.ToGRPCError(err)
	}

	return &exchangev1.CreateOrderResponse{
		OrderId: out.OrderID,
		Status:  string(out.Status),
	}, nil
}

// GetOrderStatus возвращает текущий статус существующего ордера.
func (s *Server) GetOrderStatus(ctx context.Context, req *exchangev1.GetOrderStatusRequest) (*exchangev1.GetOrderStatusResponse, error) {
	order, err := s.getOrderStatus.Execute(ctx, orderuc.GetOrderStatusInput{
		OrderID: req.GetOrderId(),
		UserID:  req.GetUserId(),
	})
	if err != nil {
		return nil, mapper.ToGRPCError(err)
	}

	return mapper.OrderToGetOrderStatusResponse(order), nil
}

// StreamOrderUpdates передаёт текущий и последующие статусы ордера.
func (s *Server) StreamOrderUpdates(req *exchangev1.StreamOrderUpdatesRequest, stream exchangev1.OrderService_StreamOrderUpdatesServer) error {
	err := s.streamOrderUpdates.Execute(stream.Context(), orderuc.StreamOrderUpdatesInput{
		OrderID: req.GetOrderId(),
		UserID:  req.GetUserId(),
	}, func(update orderuc.UpdateEvent) error {
		return stream.Send(mapper.OrderUpdateToProto(update))
	})
	if err == nil {
		return nil
	}
	if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrInvalidArgument) {
		return mapper.ToGRPCError(err)
	}
	return err
}
