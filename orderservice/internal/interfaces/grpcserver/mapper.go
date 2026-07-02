package grpcserver

import (
	"errors"
	"fmt"
	"strings"

	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
	orderv1 "github.com/exchange-grpc/proto/pb/order/v1"
	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toGRPCError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, domain.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrMarketInactive):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
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

func moneyToString(money *commonv1.Money) string {
	if money == nil {
		return ""
	}
	return strings.TrimSpace(money.GetAmount())
}

func decimalToString(decimal *commonv1.Decimal) string {
	if decimal == nil {
		return ""
	}
	return strings.TrimSpace(decimal.GetValue())
}

func uuidToString(id *commonv1.Uuid) string {
	if id == nil {
		return ""
	}
	return strings.TrimSpace(id.GetValue())
}

func stringToUuid(value string) *commonv1.Uuid {
	return &commonv1.Uuid{Value: value}
}

func moneyFromString(amount string) *commonv1.Money {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return nil
	}
	return &commonv1.Money{Amount: amount, Currency: "USD"}
}

func decimalFromString(value string) *commonv1.Decimal {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &commonv1.Decimal{Value: value}
}

func orderToGetOrderStatusResponse(order domain.Order) *orderv1.GetOrderStatusResponse {
	return &orderv1.GetOrderStatusResponse{
		OrderId:  stringToUuid(order.ID),
		UserId:   stringToUuid(order.UserID),
		MarketId: order.MarketID,
		Side:     orderSideToProto(order.Side),
		Price:    moneyFromString(order.Price),
		Quantity: decimalFromString(order.Quantity),
		Status:   orderStatusToProto(order.Status),
	}
}

func orderUpdateToProto(update application.UpdateEvent) *orderv1.OrderUpdate {
	return &orderv1.OrderUpdate{
		OrderId: stringToUuid(update.OrderID),
		Status:  orderStatusToProto(update.Status),
	}
}
