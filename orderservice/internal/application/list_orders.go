package application

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/exchange-grpc/orderservice/internal/domain"
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
func (uc *ListOrders) Execute(ctx context.Context, input ListOrdersInput) (ListOrdersOutput, error) {
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return ListOrdersOutput{}, fmt.Errorf("%w: user_id is required", domain.ErrInvalidArgument)
	}

	orders, err := uc.orders.ListByUserID(ctx, userID)
	if err != nil {
		return ListOrdersOutput{}, err
	}

	slices.SortFunc(orders, func(a, b domain.Order) int {
		return strings.Compare(a.ID, b.ID)
	})

	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = listOrdersDefaultPageSize
	}
	if pageSize > listOrdersMaxPageSize {
		pageSize = listOrdersMaxPageSize
	}

	start := 0
	if input.PageToken != "" {
		idx, found := slices.BinarySearchFunc(orders, input.PageToken, func(order domain.Order, token string) int {
			return strings.Compare(order.ID, token)
		})
		if found {
			start = idx + 1
		} else if idx < len(orders) {
			start = idx
		} else {
			return ListOrdersOutput{Orders: []domain.Order{}}, nil
		}
	}

	end := start + int(pageSize)
	hasMore := end < len(orders)
	if end > len(orders) {
		end = len(orders)
	}

	page := orders[start:end]
	var nextPageToken string
	if hasMore && len(page) > 0 {
		nextPageToken = page[len(page)-1].ID
	}

	return ListOrdersOutput{
		Orders:        page,
		NextPageToken: nextPageToken,
		HasMore:       hasMore,
	}, nil
}
