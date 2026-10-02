package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/shared/tracing"
	"go.uber.org/zap"
)

// UpdateOrderStatusInput — запрос на смену статуса ордера.
type UpdateOrderStatusInput struct {
	OrderID string
	UserID  string
	Status  domain.OrderStatus
}

// UpdateOrderStatus меняет статус ордера и пишет событие в outbox в той же транзакции.
type UpdateOrderStatus struct {
	orders domain.OrderRepository
	tx     TxManager
	outbox OutboxStore
	log    *zap.Logger
	now    func() time.Time
}

// NewUpdateOrderStatus создаёт use case UpdateOrderStatus.
// tx и outbox могут быть nil: WithinTx — passthrough, Append — no-op (тесты).
func NewUpdateOrderStatus(orders domain.OrderRepository, tx TxManager, outbox OutboxStore, log *zap.Logger) *UpdateOrderStatus {
	if log == nil {
		log = zap.NewNop()
	}
	if tx == nil {
		tx = nopTxManager{}
	}
	if outbox == nil {
		outbox = nopOutbox{}
	}
	return &UpdateOrderStatus{
		orders: orders,
		tx:     tx,
		outbox: outbox,
		log:    log,
		now:    time.Now,
	}
}

// Execute обновляет статус ордера, если он принадлежит пользователю.
// Повтор с тем же статусом — no-op. UPDATE условный по прочитанному статусу:
// если другой процесс успел сменить статус, возвращается ErrConflict без записи в outbox.
func (uc *UpdateOrderStatus) Execute(ctx context.Context, input UpdateOrderStatusInput) (err error) {
	ctx, span := tracing.Start(ctx, "order.UpdateOrderStatus",
		tracing.Attr("order.id", strings.TrimSpace(input.OrderID)),
		tracing.Attr("user_id", strings.TrimSpace(input.UserID)),
	)
	defer tracing.End(span, &err)

	orderID, err := domain.ParseUUID(input.OrderID, "order_id")
	if err != nil {
		return err
	}
	userID, err := domain.ParseUUID(input.UserID, "user_id")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(input.Status)) == "" {
		return fmt.Errorf("%w: status is required", domain.ErrInvalidArgument)
	}

	order, err := uc.orders.GetByIDAndUserID(ctx, orderID, userID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}
	if order.Status == input.Status {
		return nil
	}

	if err := domain.ValidateTransition(order.Status, input.Status); err != nil {
		return err
	}

	now := uc.now().UTC()
	if err := uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		if updateErr := uc.orders.UpdateStatus(ctx, order.ID, order.Status, input.Status, now); updateErr != nil {
			return updateErr
		}
		event, eventErr := newOrderOutboxEvent(EventTypeOrderStatusChanged, UpdateEvent{
			OrderID:   order.ID,
			UserID:    order.UserID,
			MarketID:  order.MarketID,
			Status:    input.Status,
			UpdatedAt: now,
		})
		if eventErr != nil {
			return eventErr
		}
		return uc.outbox.Append(ctx, event)
	}); err != nil {
		return fmt.Errorf("update order status: %w", err)
	}

	logAudit(ctx, uc.log, "order status updated", userID,
		zap.String("order_id", order.ID),
		zap.String("status", string(input.Status)),
	)
	return nil
}
