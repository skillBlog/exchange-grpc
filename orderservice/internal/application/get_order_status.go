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
	)
	defer tracing.End(span, &err)

	if strings.TrimSpace(input.OrderID) == "" {
		return domain.Order{}, fmt.Errorf("%w: order_id is required", domain.ErrInvalidArgument)
	}
	if strings.TrimSpace(input.UserID) == "" {
		return domain.Order{}, fmt.Errorf("%w: user_id is required", domain.ErrInvalidArgument)
	}

	return uc.orders.GetByIDAndUserID(ctx, input.OrderID, input.UserID)
}
