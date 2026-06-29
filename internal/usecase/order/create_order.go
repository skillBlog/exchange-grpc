package order

import (
	"context"

	"github.com/exchange-grpc/internal/domain"
	"github.com/google/uuid"
)

// CreateOrderInput — параметры создания ордера.
type CreateOrderInput struct {
	UserID    string
	MarketID  string
	Type      domain.OrderType
	Price     string
	Quantity  string
	UserRoles []string
}

// CreateOrderOutput возвращается после успешного создания ордера.
type CreateOrderOutput struct {
	OrderID string
	Status  domain.OrderStatus
}

// CreateOrder создаёт ордер, когда целевой рынок доступен.
type CreateOrder struct {
	orders  domain.OrderRepository
	markets MarketChecker
	hub     *UpdateHub
}

// NewCreateOrder создаёт use case CreateOrder.
func NewCreateOrder(orders domain.OrderRepository, markets MarketChecker, hub *UpdateHub) *CreateOrder {
	return &CreateOrder{
		orders:  orders,
		markets: markets,
		hub:     hub,
	}
}

// Execute проверяет рынок, создаёт ордер и сохраняет его.
func (uc *CreateOrder) Execute(ctx context.Context, input CreateOrderInput) (CreateOrderOutput, error) {
	if err := uc.markets.EnsureMarketAvailable(ctx, input.MarketID, input.UserRoles); err != nil {
		return CreateOrderOutput{}, err
	}

	order, err := domain.NewOrder(
		uuid.NewString(),
		input.UserID,
		input.MarketID,
		input.Type,
		input.Price,
		input.Quantity,
	)
	if err != nil {
		return CreateOrderOutput{}, err
	}

	if err := uc.orders.Save(ctx, order); err != nil {
		return CreateOrderOutput{}, err
	}

	if uc.hub != nil {
		uc.hub.Publish(order.ID, order.Status)
	}

	return CreateOrderOutput{
		OrderID: order.ID,
		Status:  order.Status,
	}, nil
}
