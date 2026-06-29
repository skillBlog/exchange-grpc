package order

import (
	orderuc "github.com/exchange-grpc/internal/usecase/order"
	"github.com/exchange-grpc/internal/domain"
)

// Services группирует use case'ы Order для gRPC-обработчиков.
type Services struct {
	CreateOrder        *orderuc.CreateOrder
	GetOrderStatus     *orderuc.GetOrderStatus
	StreamOrderUpdates *orderuc.StreamOrderUpdates
	UpdateOrderStatus  *orderuc.UpdateOrderStatus
	Hub                *orderuc.UpdateHub
}

// NewServices подключает use case'ы Order с общим hub обновлений.
func NewServices(orders domain.OrderRepository, markets orderuc.MarketChecker) Services {
	hub := orderuc.NewUpdateHub()
	return Services{
		Hub:                hub,
		CreateOrder:        orderuc.NewCreateOrder(orders, markets, hub),
		GetOrderStatus:     orderuc.NewGetOrderStatus(orders),
		StreamOrderUpdates: orderuc.NewStreamOrderUpdates(orders, hub),
		UpdateOrderStatus:  orderuc.NewUpdateOrderStatus(orders, hub),
	}
}
