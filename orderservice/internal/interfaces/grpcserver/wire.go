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
	CreateOrder            *application.CreateOrder
	GetOrderStatus         *application.GetOrderStatus
	ListOrders             *application.ListOrders
	StreamOrderUpdates     *application.StreamOrderUpdates
	StreamUserOrderUpdates *application.StreamUserOrderUpdates
	UpdateOrderStatus      *application.UpdateOrderStatus
	Hub                    *hub.UpdateHub
}

// NewServices подключает use case'ы Order с общим hub обновлений.
// Live-события: Kafka consumer → hub (G5). Без брокера hub пишет OutboxRelay.
func NewServices(
	orders domain.OrderRepository,
	idempotency domain.IdempotencyStore,
	markets application.MarketChecker,
	limiter application.CreateOrderRateLimiter,
	tx application.TxManager,
	outbox application.OutboxStore,
	hubBufferSize int,
	hubPublishTimeout time.Duration,
	log *zap.Logger,
) Services {
	orderHub := hub.NewUpdateHub(hubBufferSize, log, hubPublishTimeout)
	return Services{
		Hub:                    orderHub,
		CreateOrder:            application.NewCreateOrder(orders, markets, idempotency, tx, outbox, limiter, log),
		GetOrderStatus:         application.NewGetOrderStatus(orders),
		ListOrders:             application.NewListOrders(orders),
		StreamOrderUpdates:     application.NewStreamOrderUpdates(orders, orderHub),
		StreamUserOrderUpdates: application.NewStreamUserOrderUpdates(orderHub),
		UpdateOrderStatus:      application.NewUpdateOrderStatus(orders, tx, outbox, log),
	}
}
