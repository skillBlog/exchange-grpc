package grpcserver

import (
	"context"
	"errors"

	orderv1 "github.com/exchange-grpc/proto/pb/order/v1"
	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/shared/grpc"
)

// Server реализует order.v1.OrderService.
type Server struct {
	orderv1.UnimplementedOrderServiceServer
	createOrder        *application.CreateOrder
	getOrderStatus     *application.GetOrderStatus
	streamOrderUpdates *application.StreamOrderUpdates
}

// NewServer создаёт gRPC-сервер Order.
func NewServer(services Services) *Server {
	return &Server{
		createOrder:        services.CreateOrder,
		getOrderStatus:     services.GetOrderStatus,
		streamOrderUpdates: services.StreamOrderUpdates,
	}
}

// CreateOrder создаёт market-ордер после проверки целевого рынка.
func (s *Server) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	userID, ok := grpc.UserIDFromContext(ctx)
	if !ok {
		return nil, grpc.ErrMissingUserID()
	}

	side, err := orderSideToDomain(req.GetSide())
	if err != nil {
		return nil, toGRPCError(err)
	}

	out, err := s.createOrder.Execute(ctx, application.CreateOrderInput{
		UserID:    userID,
		MarketID:  req.GetMarketId(),
		Side:      side,
		Price:     moneyToString(req.GetPrice()),
		Quantity:  decimalToString(req.GetQuantity()),
		UserRoles: grpc.RolesFromContext(ctx),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &orderv1.CreateOrderResponse{
		OrderId: stringToUuid(out.OrderID),
		Status:  orderStatusToProto(out.Status),
	}, nil
}

// GetOrderStatus возвращает текущий статус существующего ордера.
func (s *Server) GetOrderStatus(ctx context.Context, req *orderv1.GetOrderStatusRequest) (*orderv1.GetOrderStatusResponse, error) {
	userID, ok := grpc.UserIDFromContext(ctx)
	if !ok {
		return nil, grpc.ErrMissingUserID()
	}

	order, err := s.getOrderStatus.Execute(ctx, application.GetOrderStatusInput{
		OrderID: uuidToString(req.GetOrderId()),
		UserID:  userID,
	})
	if err != nil {
		return nil, toGRPCError(err)
	}

	return orderToGetOrderStatusResponse(order), nil
}

// StreamOrderUpdates передаёт текущий и последующие статусы ордера.
func (s *Server) StreamOrderUpdates(req *orderv1.StreamOrderUpdatesRequest, stream orderv1.OrderService_StreamOrderUpdatesServer) error {
	userID, ok := grpc.UserIDFromContext(stream.Context())
	if !ok {
		return grpc.ErrMissingUserID()
	}

	err := s.streamOrderUpdates.Execute(stream.Context(), application.StreamOrderUpdatesInput{
		OrderID: uuidToString(req.GetOrderId()),
		UserID:  userID,
	}, func(update application.UpdateEvent) error {
		return stream.Send(orderUpdateToProto(update))
	})
	if err == nil {
		return nil
	}
	if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrInvalidArgument) {
		return toGRPCError(err)
	}
	return err
}
