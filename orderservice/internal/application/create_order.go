package application

import (
	"context"
	"strings"
	"time"

	"github.com/exchange-grpc/orderservice/internal/domain"
)

// CreateOrderInput — параметры создания ордера.
type CreateOrderInput struct {
	UserID         string
	MarketID       string
	Side           domain.OrderSide
	Price          domain.Money
	Quantity       domain.Decimal
	UserRoles      []string
	IdempotencyKey string
}

// CreateOrderOutput возвращается после успешного создания ордера.
type CreateOrderOutput struct {
	OrderID string
	Status  domain.OrderStatus
}

// CreateOrder создаёт ордер, когда целевой рынок доступен.
type CreateOrder struct {
	orders      domain.OrderRepository
	markets     MarketChecker
	idempotency domain.IdempotencyStore
	notifier    OrderNotifier
	limiter     CreateOrderRateLimiter
	now         func() time.Time
}

// NewCreateOrder создаёт use case CreateOrder.
func NewCreateOrder(
	orders domain.OrderRepository,
	markets MarketChecker,
	idempotency domain.IdempotencyStore,
	notifier OrderNotifier,
	limiter CreateOrderRateLimiter,
) *CreateOrder {
	return &CreateOrder{
		orders:      orders,
		markets:     markets,
		idempotency: idempotency,
		notifier:    notifier,
		limiter:     limiter,
		now:         time.Now,
	}
}

// Execute проверяет рынок, создаёт ордер и сохраняет его.
func (uc *CreateOrder) Execute(ctx context.Context, input CreateOrderInput) (CreateOrderOutput, error) {
	idempotencyKey := strings.TrimSpace(input.IdempotencyKey)
	if idempotencyKey != "" && uc.idempotency != nil {
		if orderID, found, err := uc.idempotency.GetOrderID(ctx, input.UserID, idempotencyKey); err != nil {
			return CreateOrderOutput{}, err
		} else if found {
			order, err := uc.orders.GetByIDAndUserID(ctx, orderID, input.UserID)
			if err != nil {
				return CreateOrderOutput{}, err
			}
			return CreateOrderOutput{OrderID: order.ID, Status: order.Status}, nil
		}
	}

	if uc.limiter != nil {
		if err := uc.limiter.Allow(ctx, input.UserID, input.UserRoles); err != nil {
			return CreateOrderOutput{}, err
		}
	}

	if err := uc.markets.EnsureMarketAvailable(ctx, input.MarketID, input.UserRoles); err != nil {
		return CreateOrderOutput{}, err
	}

	now := uc.now().UTC()
	order, err := domain.NewOrder(
		domain.NewOrderID(),
		input.UserID,
		input.MarketID,
		input.Side,
		input.Price,
		input.Quantity,
		now,
	)
	if err != nil {
		return CreateOrderOutput{}, err
	}

	if err := uc.orders.Create(ctx, order); err != nil {
		return CreateOrderOutput{}, err
	}

	if idempotencyKey != "" && uc.idempotency != nil {
		if err := uc.idempotency.Save(ctx, input.UserID, idempotencyKey, order.ID); err != nil {
			return CreateOrderOutput{}, err
		}
	}

	if uc.notifier != nil {
		uc.notifier.Publish(order.ID, order.Status)
	}

	return CreateOrderOutput{
		OrderID: order.ID,
		Status:  order.Status,
	}, nil
}
