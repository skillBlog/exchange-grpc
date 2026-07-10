package grpcserver

import (
	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"go.uber.org/zap"
)

// Services группирует use case'ы Order для gRPC-обработчиков.
type Services struct {
	CreateOrder        *application.CreateOrder
	GetOrderStatus     *application.GetOrderStatus
	ListOrders         *application.ListOrders
	StreamOrderUpdates *application.StreamOrderUpdates
	UpdateOrderStatus  *application.UpdateOrderStatus
	Hub                *application.UpdateHub
}

// NewServices подключает use case'ы Order с общим hub обновлений.
func NewServices(
	orders domain.OrderRepository,
	idempotency domain.IdempotencyStore,
	markets application.MarketChecker,
	limiter application.CreateOrderRateLimiter,
	hubBufferSize int,
	log *zap.Logger,
) Services {
	hub := application.NewUpdateHub(hubBufferSize, log)
	return Services{
		Hub:                hub,
		CreateOrder:        application.NewCreateOrder(orders, markets, idempotency, hub, limiter),
		GetOrderStatus:     application.NewGetOrderStatus(orders),
		ListOrders:         application.NewListOrders(orders),
		StreamOrderUpdates: application.NewStreamOrderUpdates(orders, hub),
		UpdateOrderStatus:  application.NewUpdateOrderStatus(orders, hub),
	}
}
