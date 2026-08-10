package application

import (
	"context"
	"strings"

	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/shared/tracing"
)

const (
	listOrdersDefaultPageSize int32 = 50
	listOrdersMaxPageSize     int32 = 100
)

// ListOrdersInput — параметры списка ордеров пользователя.
type ListOrdersInput struct {
	UserID    string
	PageToken string
	PageSize  int32
}

// ListOrdersOutput — результат курсорной выборки ордеров.
type ListOrdersOutput struct {
	Orders        []domain.Order
	NextPageToken string
	HasMore       bool
}

// ListOrders возвращает ордера текущего пользователя.
type ListOrders struct {
	orders domain.OrderRepository
}

// NewListOrders создаёт use case ListOrders.
func NewListOrders(orders domain.OrderRepository) *ListOrders {
	return &ListOrders{orders: orders}
}

// Execute возвращает страницу ордеров пользователя.
func (uc *ListOrders) Execute(ctx context.Context, input ListOrdersInput) (out ListOrdersOutput, err error) {
	ctx, span := tracing.Start(ctx, "order.ListOrders")
	defer tracing.End(span, &err)

	userID, err := domain.ParseUUID(input.UserID, "user_id")
	if err != nil {
		return ListOrdersOutput{}, err
	}

	pageToken := strings.TrimSpace(input.PageToken)
	if pageToken != "" {
		pageToken, err = domain.ParseUUID(pageToken, "page_token")
		if err != nil {
			return ListOrdersOutput{}, err
		}
	}

	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = listOrdersDefaultPageSize
	}
	if pageSize > listOrdersMaxPageSize {
		pageSize = listOrdersMaxPageSize
	}

	// Запрашиваем pageSize+1, чтобы понять, есть ли следующая страница.
	orders, err := uc.orders.ListByUserID(ctx, userID, int(pageSize)+1, pageToken)
	if err != nil {
		return ListOrdersOutput{}, err
	}

	hasMore := len(orders) > int(pageSize)
	if hasMore {
		orders = orders[:pageSize]
	}

	var nextPageToken string
	if hasMore && len(orders) > 0 {
		nextPageToken = orders[len(orders)-1].ID
	}

	return ListOrdersOutput{
		Orders:        orders,
		NextPageToken: nextPageToken,
		HasMore:       hasMore,
	}, nil
}
