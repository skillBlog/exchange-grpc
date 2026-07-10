package grpcserver

import (
	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
	orderv1 "github.com/exchange-grpc/proto/pb/order/v1"
	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
)

// Mapper преобразует protobuf-запросы в application input/output.
type Mapper struct{}

func (Mapper) CreateOrderRequestToInput(req *orderv1.CreateOrderRequest, userID string, roles []string) (application.CreateOrderInput, error) {
	price, err := moneyToDomain(req.GetPrice())
	if err != nil {
		return application.CreateOrderInput{}, err
	}
	quantity, err := decimalToDomain(req.GetQuantity())
	if err != nil {
		return application.CreateOrderInput{}, err
	}
	side, err := orderSideToDomain(req.GetSide())
	if err != nil {
		return application.CreateOrderInput{}, err
	}

	return application.CreateOrderInput{
		UserID:         userID,
		MarketID:       req.GetMarketId(),
		Side:           side,
		Price:          price,
		Quantity:       quantity,
		UserRoles:      roles,
		IdempotencyKey: req.GetIdempotencyKey(),
	}, nil
}

func (Mapper) ListOrdersRequestToInput(req *orderv1.ListOrdersRequest, userID string) application.ListOrdersInput {
	input := application.ListOrdersInput{UserID: userID}
	if pagination := req.GetPagination(); pagination != nil {
		input.PageToken = pagination.GetPageToken()
		input.PageSize = pagination.GetPageSize()
	}
	return input
}

func (Mapper) ListOrdersOutputToResponse(out application.ListOrdersOutput) *orderv1.ListOrdersResponse {
	orders := make([]*orderv1.OrderSummary, 0, len(out.Orders))
	for _, order := range out.Orders {
		orders = append(orders, orderToSummary(order))
	}
	return &orderv1.ListOrdersResponse{
		Orders: orders,
		PageInfo: &commonv1.CursorPageInfo{
			NextPageToken: out.NextPageToken,
			HasMore:       out.HasMore,
		},
	}
}

func orderToSummary(order domain.Order) *orderv1.OrderSummary {
	return &orderv1.OrderSummary{
		OrderId:  stringToUuid(order.ID),
		MarketId: order.MarketID,
		Side:     orderSideToProto(order.Side),
		Price:    moneyToProto(order.Price),
		Quantity: decimalToProto(order.Quantity),
		Status:   orderStatusToProto(order.Status),
	}
}
