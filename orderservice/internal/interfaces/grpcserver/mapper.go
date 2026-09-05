package grpcserver

import (
	"fmt"
	"strings"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
	orderv1 "github.com/exchange-grpc/proto/pb/order/v1"
	"github.com/exchange-grpc/shared/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toGRPCError(err error) error {
	return grpc.ToStatusError(err, "too many requests")
}

func orderSideToDomain(side commonv1.OrderSide) (domain.OrderSide, error) {
	switch side {
	case commonv1.OrderSide_ORDER_SIDE_BUY:
		return domain.OrderSideBuy, nil
	case commonv1.OrderSide_ORDER_SIDE_SELL:
		return domain.OrderSideSell, nil
	default:
		return "", fmt.Errorf("%w: unsupported order side %v", domain.ErrInvalidArgument, side)
	}
}

func orderSideToProto(side domain.OrderSide) commonv1.OrderSide {
	switch side {
	case domain.OrderSideBuy:
		return commonv1.OrderSide_ORDER_SIDE_BUY
	case domain.OrderSideSell:
		return commonv1.OrderSide_ORDER_SIDE_SELL
	default:
		return commonv1.OrderSide_ORDER_SIDE_UNSPECIFIED
	}
}

func orderStatusToProto(status domain.OrderStatus) commonv1.OrderStatus {
	switch status {
	case domain.OrderStatusCreated:
		return commonv1.OrderStatus_ORDER_STATUS_CREATED
	case domain.OrderStatusFilled:
		return commonv1.OrderStatus_ORDER_STATUS_FILLED
	case domain.OrderStatusRejected:
		return commonv1.OrderStatus_ORDER_STATUS_REJECTED
	case domain.OrderStatusFailed:
		return commonv1.OrderStatus_ORDER_STATUS_FAILED
	case domain.OrderStatusCancelled:
		return commonv1.OrderStatus_ORDER_STATUS_CANCELLED
	default:
		return commonv1.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

func orderStatusFromProto(status commonv1.OrderStatus) (domain.OrderStatus, error) {
	switch status {
	case commonv1.OrderStatus_ORDER_STATUS_UNSPECIFIED:
		return "", nil
	case commonv1.OrderStatus_ORDER_STATUS_CREATED:
		return domain.OrderStatusCreated, nil
	case commonv1.OrderStatus_ORDER_STATUS_FILLED:
		return domain.OrderStatusFilled, nil
	case commonv1.OrderStatus_ORDER_STATUS_REJECTED:
		return domain.OrderStatusRejected, nil
	case commonv1.OrderStatus_ORDER_STATUS_FAILED:
		return domain.OrderStatusFailed, nil
	case commonv1.OrderStatus_ORDER_STATUS_CANCELLED:
		return domain.OrderStatusCancelled, nil
	default:
		return "", fmt.Errorf("%w: unsupported order status %v", domain.ErrInvalidArgument, status)
	}
}

func moneyToDomain(money *commonv1.Money) (domain.Money, error) {
	if money == nil {
		return domain.Money{}, nil
	}
	return domain.NewMoney(money.GetAmount(), money.GetCurrency())
}

func decimalToDomain(decimal *commonv1.Decimal) (domain.Decimal, error) {
	if decimal == nil {
		return domain.Decimal{}, fmt.Errorf("%w: quantity is required", domain.ErrInvalidArgument)
	}
	return domain.NewDecimal(decimal.GetValue())
}

func moneyToProto(money domain.Money) *commonv1.Money {
	if money.IsZero() {
		return nil
	}
	return &commonv1.Money{Amount: money.Amount, Currency: money.Currency}
}

func decimalToProto(decimal domain.Decimal) *commonv1.Decimal {
	return &commonv1.Decimal{Value: decimal.Value}
}

func orderToGetOrderStatusResponse(order domain.Order) *orderv1.GetOrderStatusResponse {
	return &orderv1.GetOrderStatusResponse{
		OrderId:   order.ID,
		UserId:    order.UserID,
		MarketId:  order.MarketID,
		Side:      orderSideToProto(order.Side),
		Price:     moneyToProto(order.Price),
		Quantity:  decimalToProto(order.Quantity),
		Status:    orderStatusToProto(order.Status),
		CreatedAt: timestamppb.New(order.CreatedAt),
		UpdatedAt: timestamppb.New(order.UpdatedAt),
	}
}

func orderUpdateToProto(update application.UpdateEvent) *orderv1.OrderUpdate {
	msg := &orderv1.OrderUpdate{
		OrderId: update.OrderID,
		Status:  orderStatusToProto(update.Status),
	}
	if !update.UpdatedAt.IsZero() {
		msg.UpdatedAt = timestamppb.New(update.UpdatedAt)
	}
	return msg
}

func normalizeOrderID(orderID string) string {
	return strings.TrimSpace(orderID)
}
