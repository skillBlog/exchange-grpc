package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/shared/tracing"
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

// Execute проверяет рынок, резервирует idempotency-ключ и создаёт ордер.
func (uc *CreateOrder) Execute(ctx context.Context, input CreateOrderInput) (out CreateOrderOutput, err error) {
	ctx, span := tracing.Start(ctx, "order.CreateOrder",
		tracing.Attr("market.id", strings.TrimSpace(input.MarketID)),
	)
	defer tracing.End(span, &err)

	userID := strings.TrimSpace(input.UserID)
	marketID := strings.TrimSpace(input.MarketID)
	if userID == "" {
		return CreateOrderOutput{}, fmt.Errorf("%w: user_id is required", domain.ErrInvalidArgument)
	}
	if marketID == "" {
		return CreateOrderOutput{}, fmt.Errorf("%w: market_id is required", domain.ErrInvalidArgument)
	}
	input.UserID = userID
	input.MarketID = marketID
	span.SetAttributes(tracing.Attr("market.id", marketID))

	idempotencyKey := strings.TrimSpace(input.IdempotencyKey)
	if idempotencyKey != "" && uc.idempotency != nil {
		if orderID, found, getErr := uc.idempotency.GetOrderID(ctx, input.UserID, idempotencyKey); getErr != nil {
			return CreateOrderOutput{}, getErr
		} else if found {
			return uc.existingOrder(ctx, orderID, input.UserID)
		}
	}

	if uc.limiter != nil {
		if err = uc.limiter.Allow(ctx, input.UserID, input.UserRoles); err != nil {
			return CreateOrderOutput{}, err
		}
	}

	if err = uc.markets.EnsureMarketAvailable(ctx, input.MarketID, input.UserRoles); err != nil {
		return CreateOrderOutput{}, err
	}

	now := uc.now().UTC()
	orderID := domain.NewOrderID()
	order, err := domain.NewOrder(
		orderID,
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

	if idempotencyKey != "" && uc.idempotency != nil {
		reserved, existingOrderID, reserveErr := uc.idempotency.Reserve(ctx, input.UserID, idempotencyKey, order.ID)
		if reserveErr != nil {
			return CreateOrderOutput{}, reserveErr
		}
		if !reserved {
			return uc.existingOrder(ctx, existingOrderID, input.UserID)
		}
	}

	if err = uc.orders.Create(ctx, order); err != nil {
		if idempotencyKey != "" && uc.idempotency != nil {
			_ = uc.idempotency.Release(ctx, input.UserID, idempotencyKey)
		}
		return CreateOrderOutput{}, err
	}

	if uc.notifier != nil {
		uc.notifier.Publish(order.ID, order.Status)
	}

	return CreateOrderOutput{
		OrderID: order.ID,
		Status:  order.Status,
	}, nil
}

func (uc *CreateOrder) existingOrder(ctx context.Context, orderID, userID string) (CreateOrderOutput, error) {
	order, err := uc.orders.GetByIDAndUserID(ctx, orderID, userID)
	if err != nil {
		return CreateOrderOutput{}, err
	}
	return CreateOrderOutput{OrderID: order.ID, Status: order.Status}, nil
}
