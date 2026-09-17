package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/shared/tracing"
)

// GetOrderStatusInput идентифицирует ордер для запроса статуса.
type GetOrderStatusInput struct {
	OrderID string
	UserID  string
}

// GetOrderStatus возвращает текущее состояние ордера пользователя.
type GetOrderStatus struct {
	orders domain.OrderRepository
}

// NewGetOrderStatus создаёт use case GetOrderStatus.
func NewGetOrderStatus(orders domain.OrderRepository) *GetOrderStatus {
	return &GetOrderStatus{orders: orders}
}

// Execute загружает ордер при совпадении order_id и user_id.
func (uc *GetOrderStatus) Execute(ctx context.Context, input GetOrderStatusInput) (order domain.Order, err error) {
	ctx, span := tracing.Start(ctx, "order.GetOrderStatus",
		tracing.Attr("order.id", strings.TrimSpace(input.OrderID)),
		tracing.Attr("user_id", strings.TrimSpace(input.UserID)),
	)
	defer tracing.End(span, &err)

	orderID, err := domain.ParseUUID(input.OrderID, "order_id")
	if err != nil {
		return domain.Order{}, err
	}
	userID, err := domain.ParseUUID(input.UserID, "user_id")
	if err != nil {
		return domain.Order{}, err
	}

	order, err = uc.orders.GetByIDAndUserID(ctx, orderID, userID)
	if err != nil {
		return domain.Order{}, fmt.Errorf("get order: %w", err)
	}
	return order, nil
}
