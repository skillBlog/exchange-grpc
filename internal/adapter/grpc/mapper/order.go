package mapper

import (
	"fmt"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	"github.com/exchange-grpc/internal/domain"
	orderuc "github.com/exchange-grpc/internal/usecase/order"
)

// OrderTypeToDomain преобразует тип ордера из protobuf в доменное значение.
func OrderTypeToDomain(orderType exchangev1.OrderType) (domain.OrderType, error) {
	switch orderType {
	case exchangev1.OrderType_ORDER_TYPE_LIMIT:
		return domain.OrderTypeLimit, nil
	case exchangev1.OrderType_ORDER_TYPE_MARKET:
		return domain.OrderTypeMarket, nil
	default:
		return "", fmt.Errorf("%w: unsupported order type %v", domain.ErrInvalidArgument, orderType)
	}
}

// OrderTypeToProto преобразует доменный тип ордера в protobuf.
func OrderTypeToProto(orderType domain.OrderType) exchangev1.OrderType {
	switch orderType {
	case domain.OrderTypeLimit:
		return exchangev1.OrderType_ORDER_TYPE_LIMIT
	case domain.OrderTypeMarket:
		return exchangev1.OrderType_ORDER_TYPE_MARKET
	default:
		return exchangev1.OrderType_ORDER_TYPE_UNSPECIFIED
	}
}

// OrderToGetOrderStatusResponse преобразует доменный ордер в gRPC-ответ.
func OrderToGetOrderStatusResponse(order domain.Order) *exchangev1.GetOrderStatusResponse {
	return &exchangev1.GetOrderStatusResponse{
		OrderId:   order.ID,
		UserId:    order.UserID,
		MarketId:  order.MarketID,
		OrderType: OrderTypeToProto(order.Type),
		Price:     order.Price,
		Quantity:  order.Quantity,
		Status:    string(order.Status),
	}
}

// OrderUpdateToProto преобразует событие обновления ордера в protobuf.
func OrderUpdateToProto(update orderuc.UpdateEvent) *exchangev1.OrderUpdate {
	return &exchangev1.OrderUpdate{
		OrderId: update.OrderID,
		Status:  string(update.Status),
	}
}
