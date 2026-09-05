package grpcserver

import (
	"context"
	"errors"

	"github.com/exchange-grpc/orderservice/internal/application"
	orderv1 "github.com/exchange-grpc/proto/pb/order/v1"
	"github.com/exchange-grpc/shared/grpc"
)

// Server реализует order.v1.OrderService.
type Server struct {
	orderv1.UnimplementedOrderServiceServer
	mapper                 Mapper
	createOrder            *application.CreateOrder
	getOrderStatus         *application.GetOrderStatus
	listOrders             *application.ListOrders
	streamOrderUpdates     *application.StreamOrderUpdates
	streamUserOrderUpdates *application.StreamUserOrderUpdates
}

// NewServer создаёт gRPC-сервер Order.
func NewServer(services Services) *Server {
	return &Server{
		createOrder:            services.CreateOrder,
		getOrderStatus:         services.GetOrderStatus,
		listOrders:             services.ListOrders,
		streamOrderUpdates:     services.StreamOrderUpdates,
		streamUserOrderUpdates: services.StreamUserOrderUpdates,
	}
}

// CreateOrder создаёт market-ордер после проверки целевого рынка.
func (s *Server) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	userID, ok := grpc.UserIDFromContext(ctx)
	if !ok {
		return nil, grpc.ErrMissingUserID()
	}

	input, err := s.mapper.CreateOrderRequestToInput(req, userID, grpc.RolesFromContext(ctx))
	if err != nil {
		return nil, toGRPCError(err)
	}

	out, err := s.createOrder.Execute(ctx, input)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &orderv1.CreateOrderResponse{
		OrderId: out.OrderID,
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
		OrderID: normalizeOrderID(req.GetOrderId()),
		UserID:  userID,
	})
	if err != nil {
		return nil, toGRPCError(err)
	}

	return orderToGetOrderStatusResponse(order), nil
}

// ListOrders возвращает список ордеров текущего пользователя.
func (s *Server) ListOrders(ctx context.Context, req *orderv1.ListOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	userID, ok := grpc.UserIDFromContext(ctx)
	if !ok {
		return nil, grpc.ErrMissingUserID()
	}

	input, err := s.mapper.ListOrdersRequestToInput(req, userID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	out, err := s.listOrders.Execute(ctx, input)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return s.mapper.ListOrdersOutputToResponse(out), nil
}

// StreamOrderUpdates передаёт текущий и последующие статусы ордера.
func (s *Server) StreamOrderUpdates(req *orderv1.StreamOrderUpdatesRequest, stream orderv1.OrderService_StreamOrderUpdatesServer) error {
	userID, ok := grpc.UserIDFromContext(stream.Context())
	if !ok {
		return grpc.ErrMissingUserID()
	}

	err := s.streamOrderUpdates.Execute(stream.Context(), application.StreamOrderUpdatesInput{
		OrderID: normalizeOrderID(req.GetOrderId()),
		UserID:  userID,
	}, func(update application.UpdateEvent) error {
		return stream.Send(orderUpdateToProto(update))
	})
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return toGRPCError(err)
}

// StreamUserOrderUpdates передаёт обновления всех ордеров текущего пользователя.
func (s *Server) StreamUserOrderUpdates(req *orderv1.StreamUserOrderUpdatesRequest, stream orderv1.OrderService_StreamUserOrderUpdatesServer) error {
	userID, ok := grpc.UserIDFromContext(stream.Context())
	if !ok {
		return grpc.ErrMissingUserID()
	}

	input, err := s.mapper.StreamUserOrderUpdatesRequestToInput(req, userID)
	if err != nil {
		return toGRPCError(err)
	}

	err = s.streamUserOrderUpdates.Execute(stream.Context(), input, func(update application.UpdateEvent) error {
		return stream.Send(orderUpdateToProto(update))
	})
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return toGRPCError(err)
}
