package grpcserver

import (
	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
)

// Services группирует use case'ы Order для gRPC-обработчиков.
type Services struct {
	CreateOrder        *application.CreateOrder
	GetOrderStatus     *application.GetOrderStatus
	StreamOrderUpdates *application.StreamOrderUpdates
	UpdateOrderStatus  *application.UpdateOrderStatus
	Hub                *application.UpdateHub
}

// NewServices подключает use case'ы Order с общим hub обновлений.
func NewServices(orders domain.OrderRepository, markets application.MarketChecker, hubBufferSize int) Services {
	hub := application.NewUpdateHub(hubBufferSize)
	return Services{
		Hub:                hub,
		CreateOrder:        application.NewCreateOrder(orders, markets, hub),
		GetOrderStatus:     application.NewGetOrderStatus(orders),
		StreamOrderUpdates: application.NewStreamOrderUpdates(orders, hub),
		UpdateOrderStatus:  application.NewUpdateOrderStatus(orders, hub),
	}
}
