package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/shared/tracing"
	"go.uber.org/zap"
)

const idempotencyCleanupTimeout = 3 * time.Second

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
	tx          TxManager
	outbox      OutboxStore
	limiter     CreateOrderRateLimiter
	log         *zap.Logger
	now         func() time.Time
}

// NewCreateOrder создаёт use case CreateOrder.
// tx и outbox могут быть nil: тогда WithinTx — passthrough, Append — no-op (тесты).
func NewCreateOrder(
	orders domain.OrderRepository,
	markets MarketChecker,
	idempotency domain.IdempotencyStore,
	tx TxManager,
	outbox OutboxStore,
	limiter CreateOrderRateLimiter,
	log *zap.Logger,
) *CreateOrder {
	if log == nil {
		log = zap.NewNop()
	}
	if tx == nil {
		tx = nopTxManager{}
	}
	if outbox == nil {
		outbox = nopOutbox{}
	}
	return &CreateOrder{
		orders:      orders,
		markets:     markets,
		idempotency: idempotency,
		tx:          tx,
		outbox:      outbox,
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
		if err = tracing.Run(ctx, "order.CreateOrder.getIdempotency", func(ctx context.Context) error {
			var getErr error
			orderID, found, getErr = uc.idempotency.GetOrderID(ctx, input.UserID, idempotencyKey)
			return getErr
		}); err != nil {
			return CreateOrderOutput{}, fmt.Errorf("get idempotency key: %w", err)
		} else if found {
			return uc.existingOrder(ctx, orderID, input.UserID)
		}
	}

	var limits domain.MarketLimits
	if err = tracing.Run(ctx, "order.CreateOrder.ensureMarket", func(ctx context.Context) error {
		var ensureErr error
		limits, ensureErr = uc.markets.EnsureMarketAvailable(ctx, input.MarketID, input.UserRoles)
		return ensureErr
	}); err != nil {
		return CreateOrderOutput{}, fmt.Errorf("ensure market: %w", err)
	}
	if err = domain.ValidateOrderAgainstMarket(input.Quantity, input.Price, limits); err != nil {
		return CreateOrderOutput{}, err
	}

	if uc.limiter != nil {
		if err = tracing.Run(ctx, "order.CreateOrder.rateLimit", func(ctx context.Context) error {
			return uc.limiter.Allow(ctx, input.UserID, input.UserRoles)
		}); err != nil {
			return CreateOrderOutput{}, fmt.Errorf("create order rate limit: %w", err)
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
		if err = tracing.Run(ctx, "order.CreateOrder.reserveIdempotency", func(ctx context.Context) error {
			var reserveErr error
			reserved, existingOrderID, reserveErr = uc.idempotency.Reserve(ctx, input.UserID, idempotencyKey, order.ID)
			return reserveErr
		}); err != nil {
			return CreateOrderOutput{}, fmt.Errorf("reserve idempotency key: %w", err)
		}
		if !reserved {
			out, existingErr := uc.existingOrder(ctx, existingOrderID, input.UserID)
			if errors.Is(existingErr, domain.ErrNotFound) {
				return CreateOrderOutput{}, fmt.Errorf("%w: idempotency key is already in progress", domain.ErrFailedPrecondition)
			}
			return out, existingErr
		}
	}

	if err = tracing.Run(ctx, "order.CreateOrder.insert", func(ctx context.Context) error {
		return uc.tx.WithinTx(ctx, func(ctx context.Context) error {
			if createErr := uc.orders.Create(ctx, order); createErr != nil {
				return createErr
			}
			event, eventErr := newOrderOutboxEvent(EventTypeOrderCreated, UpdateEvent{
				OrderID:   order.ID,
				UserID:    order.UserID,
				MarketID:  order.MarketID,
				Status:    order.Status,
				UpdatedAt: order.UpdatedAt,
			})
			if eventErr != nil {
				return eventErr
			}
			return uc.outbox.Append(ctx, event)
		})
	}); err != nil {
		if idempotencyKey != "" && uc.idempotency != nil {
			failCtx, cancelFail := context.WithTimeout(context.WithoutCancel(ctx), idempotencyCleanupTimeout)
			failErr := uc.idempotency.Fail(failCtx, input.UserID, idempotencyKey)
			cancelFail()
			if failErr != nil {
				logError(ctx, uc.log, "idempotency fail after order create error", input.UserID,
					zap.Error(failErr),
				)
			}
		}
		return CreateOrderOutput{}, fmt.Errorf("create order: %w", err)
	}

	if idempotencyKey != "" && uc.idempotency != nil {
		completeErr := tracing.Run(ctx, "order.CreateOrder.completeIdempotency", func(ctx context.Context) error {
			return uc.idempotency.Complete(ctx, input.UserID, idempotencyKey)
		})
		if completeErr != nil {
			logError(ctx, uc.log, "idempotency complete failed after order create", order.UserID,
				zap.String("order_id", order.ID),
				zap.Error(completeErr),
			)
		}
	}

	logAudit(ctx, uc.log, "order created", order.UserID,
		zap.String("order_id", order.ID),
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
		return CreateOrderOutput{}, fmt.Errorf("get order: %w", err)
	}
	logAudit(ctx, uc.log, "order create idempotent hit", userID,
		zap.String("order_id", order.ID),
		zap.String("status", string(order.Status)),
	)
	return CreateOrderOutput{OrderID: order.ID, Status: order.Status}, nil
}
