package grpcserver

import (
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/hub"
	"go.uber.org/zap"
)

// Services группирует use case'ы Order для gRPC-обработчиков.
type Services struct {
	CreateOrder        *application.CreateOrder
	GetOrderStatus     *application.GetOrderStatus
	ListOrders         *application.ListOrders
	StreamOrderUpdates *application.StreamOrderUpdates
	UpdateOrderStatus  *application.UpdateOrderStatus
	Hub                *hub.UpdateHub
}

// NewServices подключает use case'ы Order с общим hub обновлений.
func NewServices(
	orders domain.OrderRepository,
	idempotency domain.IdempotencyStore,
	markets application.MarketChecker,
	limiter application.CreateOrderRateLimiter,
	hubBufferSize int,
	hubPublishTimeout time.Duration,
	log *zap.Logger,
) Services {
	orderHub := hub.NewUpdateHub(hubBufferSize, log, hubPublishTimeout)
	return Services{
		Hub:                orderHub,
		CreateOrder:        application.NewCreateOrder(orders, markets, idempotency, orderHub, limiter, log),
		GetOrderStatus:     application.NewGetOrderStatus(orders),
		ListOrders:         application.NewListOrders(orders),
		StreamOrderUpdates: application.NewStreamOrderUpdates(orders, orderHub),
		UpdateOrderStatus:  application.NewUpdateOrderStatus(orders, orderHub),
	}
}
