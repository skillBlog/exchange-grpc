package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/shared/logger"
	"github.com/exchange-grpc/shared/tracing"
	"go.uber.org/zap"
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
	log         *zap.Logger
	now         func() time.Time
}

// NewCreateOrder создаёт use case CreateOrder.
func NewCreateOrder(
	orders domain.OrderRepository,
	markets MarketChecker,
	idempotency domain.IdempotencyStore,
	notifier OrderNotifier,
	limiter CreateOrderRateLimiter,
	log *zap.Logger,
) *CreateOrder {
	if log == nil {
		log = zap.NewNop()
	}
	return &CreateOrder{
		orders:      orders,
		markets:     markets,
		idempotency: idempotency,
		notifier:    notifier,
		limiter:     limiter,
		log:         log,
		now:         time.Now,
	}
}

// Execute проверяет рынок, резервирует idempotency-ключ и создаёт ордер.
func (uc *CreateOrder) Execute(ctx context.Context, input CreateOrderInput) (out CreateOrderOutput, err error) {
	ctx, span := tracing.Start(ctx, "order.CreateOrder",
		tracing.Attr("market.id", strings.TrimSpace(input.MarketID)),
		tracing.Attr("user_id", strings.TrimSpace(input.UserID)),
	)
	defer tracing.End(span, &err)

	userID := strings.TrimSpace(input.UserID)
	marketID := strings.TrimSpace(input.MarketID)
	if _, err = domain.ParseUUID(userID, "user_id"); err != nil {
		return CreateOrderOutput{}, err
	}
	if marketID == "" {
		return CreateOrderOutput{}, fmt.Errorf("%w: market_id is required", domain.ErrInvalidArgument)
	}
	input.UserID = userID
	input.MarketID = marketID

	idempotencyKey := strings.TrimSpace(input.IdempotencyKey)
	if idempotencyKey != "" && uc.idempotency != nil {
		var (
			orderID string
			found   bool
		)
		if err = runChildSpan(ctx, "order.CreateOrder.getIdempotency", func(ctx context.Context) error {
			var getErr error
			orderID, found, getErr = uc.idempotency.GetOrderID(ctx, input.UserID, idempotencyKey)
			return getErr
		}); err != nil {
			return CreateOrderOutput{}, err
		} else if found {
			return uc.existingOrder(ctx, orderID, input.UserID)
		}
	}

	var limits domain.MarketLimits
	if err = runChildSpan(ctx, "order.CreateOrder.ensureMarket", func(ctx context.Context) error {
		var ensureErr error
		limits, ensureErr = uc.markets.EnsureMarketAvailable(ctx, input.MarketID, input.UserRoles)
		return ensureErr
	}); err != nil {
		return CreateOrderOutput{}, err
	}
	if err = domain.ValidateOrderAgainstMarket(input.Quantity, input.Price, limits); err != nil {
		return CreateOrderOutput{}, err
	}

	if uc.limiter != nil {
		if err = runChildSpan(ctx, "order.CreateOrder.rateLimit", func(ctx context.Context) error {
			return uc.limiter.Allow(ctx, input.UserID, input.UserRoles)
		}); err != nil {
			return CreateOrderOutput{}, err
		}
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
		var (
			reserved        bool
			existingOrderID string
		)
		if err = runChildSpan(ctx, "order.CreateOrder.reserveIdempotency", func(ctx context.Context) error {
			var reserveErr error
			reserved, existingOrderID, reserveErr = uc.idempotency.Reserve(ctx, input.UserID, idempotencyKey, order.ID)
			return reserveErr
		}); err != nil {
			return CreateOrderOutput{}, err
		}
		if !reserved {
			out, existingErr := uc.existingOrder(ctx, existingOrderID, input.UserID)
			if errors.Is(existingErr, domain.ErrNotFound) {
				return CreateOrderOutput{}, fmt.Errorf("%w: idempotency key is already in progress", domain.ErrFailedPrecondition)
			}
			return out, existingErr
		}
	}

	if err = runChildSpan(ctx, "order.CreateOrder.insert", func(ctx context.Context) error {
		return uc.orders.Create(ctx, order)
	}); err != nil {
		if idempotencyKey != "" && uc.idempotency != nil {
			_ = uc.idempotency.Fail(ctx, input.UserID, idempotencyKey)
		}
		return CreateOrderOutput{}, err
	}

	if idempotencyKey != "" && uc.idempotency != nil {
		completeErr := runChildSpan(ctx, "order.CreateOrder.completeIdempotency", func(ctx context.Context) error {
			return uc.idempotency.Complete(ctx, input.UserID, idempotencyKey)
		})
		if completeErr != nil {
			logger.WithTrace(ctx, uc.log).Error("idempotency complete failed after order create",
				zap.String("order_id", order.ID),
				zap.String("user_id", order.UserID),
				zap.String("idempotency_key", idempotencyKey),
				zap.Error(completeErr),
			)
		}
	}

	if uc.notifier != nil {
		uc.notifier.Publish(UpdateEvent{
			OrderID:   order.ID,
			UserID:    order.UserID,
			MarketID:  order.MarketID,
			Status:    order.Status,
			UpdatedAt: order.UpdatedAt,
		})
	}

	logger.WithTrace(ctx, uc.log).Info("order created",
		zap.String("order_id", order.ID),
		zap.String("user_id", order.UserID),
		zap.String("market_id", order.MarketID),
		zap.String("side", string(order.Side)),
		zap.String("status", string(order.Status)),
		zap.String("quantity", order.Quantity.Value),
	)

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
	logger.WithTrace(ctx, uc.log).Info("order create idempotent hit",
		zap.String("order_id", order.ID),
		zap.String("user_id", userID),
		zap.String("status", string(order.Status)),
	)
	return CreateOrderOutput{OrderID: order.ID, Status: order.Status}, nil
}

func runChildSpan(ctx context.Context, name string, fn func(context.Context) error) (err error) {
	ctx, span := tracing.Start(ctx, name)
	defer tracing.End(span, &err)
	return fn(ctx)
}
