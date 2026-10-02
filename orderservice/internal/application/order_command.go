package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/exchange-grpc/orderservice/internal/domain"
)

const defaultCommandMaxAttempts = 5

// OrderCommand — входящая команда смены статуса (топик order.commands).
type OrderCommand struct {
	CommandID string
	OrderID   string
	UserID    string
	Status    domain.OrderStatus
}

type orderCommandJSON struct {
	CommandID string `json:"command_id"`
	EventID   string `json:"event_id"`
	OrderID   string `json:"order_id"`
	UserID    string `json:"user_id"`
	Status    string `json:"status"`
}

// ApplyOrderCommand пишет inbox, меняет статус и кладёт outbox-событие в одной транзакции.
type ApplyOrderCommand struct {
	tx     TxManager
	inbox  InboxStore
	update *UpdateOrderStatus
}

// NewApplyOrderCommand создаёт обработчик inbound-команд.
// tx/inbox могут быть nil: WithinTx — passthrough, inbox — всегда «новый» (тесты без дедупа).
func NewApplyOrderCommand(tx TxManager, inbox InboxStore, update *UpdateOrderStatus) *ApplyOrderCommand {
	if tx == nil {
		tx = nopTxManager{}
	}
	if inbox == nil {
		inbox = nopInbox{}
	}
	return &ApplyOrderCommand{tx: tx, inbox: inbox, update: update}
}

// Handle реализует OrderCommandHandler.
func (uc *ApplyOrderCommand) Handle(ctx context.Context, cmd OrderCommand) error {
	return uc.Execute(ctx, cmd)
}

// Execute вставляет command_id в inbox; дубль — успешный no-op без повторного UpdateStatus.
func (uc *ApplyOrderCommand) Execute(ctx context.Context, cmd OrderCommand) error {
	commandID, err := domain.ParseUUID(cmd.CommandID, "command_id")
	if err != nil {
		return err
	}

	return uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		inserted, err := uc.inbox.InsertIfNew(ctx, commandID)
		if err != nil {
			return fmt.Errorf("inbox insert: %w", err)
		}
		if !inserted {
			return nil
		}
		if uc.update == nil {
			return fmt.Errorf("%w: order command handler is not configured", domain.ErrFailedPrecondition)
		}
		return uc.update.Execute(ctx, UpdateOrderStatusInput{
			OrderID: cmd.OrderID,
			UserID:  cmd.UserID,
			Status:  cmd.Status,
		})
	})
}

// ParseOrderCommand разбирает JSON команды. command_id и event_id — синонимы.
func ParseOrderCommand(raw []byte) (OrderCommand, error) {
	var body orderCommandJSON
	if err := json.Unmarshal(raw, &body); err != nil {
		return OrderCommand{}, fmt.Errorf("%w: decode order command: %v", domain.ErrInvalidArgument, err)
	}

	idRaw := strings.TrimSpace(body.CommandID)
	if idRaw == "" {
		idRaw = strings.TrimSpace(body.EventID)
	}
	commandID, err := domain.ParseUUID(idRaw, "command_id")
	if err != nil {
		if strings.TrimSpace(idRaw) == "" {
			return OrderCommand{}, fmt.Errorf("%w: command_id is required", domain.ErrInvalidArgument)
		}
		return OrderCommand{}, err
	}
	orderID, err := domain.ParseUUID(body.OrderID, "order_id")
	if err != nil {
		return OrderCommand{}, err
	}
	userID, err := domain.ParseUUID(body.UserID, "user_id")
	if err != nil {
		return OrderCommand{}, err
	}
	status, err := parseOrderCommandStatus(body.Status)
	if err != nil {
		return OrderCommand{}, err
	}

	return OrderCommand{
		CommandID: commandID,
		OrderID:   orderID,
		UserID:    userID,
		Status:    status,
	}, nil
}

func parseOrderCommandStatus(raw string) (domain.OrderStatus, error) {
	status := domain.OrderStatus(strings.TrimSpace(raw))
	switch status {
	case domain.OrderStatusCreated,
		domain.OrderStatusFilled,
		domain.OrderStatusRejected,
		domain.OrderStatusFailed,
		domain.OrderStatusCancelled:
		return status, nil
	case "":
		return "", fmt.Errorf("%w: status is required", domain.ErrInvalidArgument)
	default:
		return "", fmt.Errorf("%w: unsupported status %q", domain.ErrInvalidArgument, raw)
	}
}

// IsPermanentCommandError — яд: ретраить нет смысла, сразу DLQ.
func IsPermanentCommandError(err error) bool {
	return errors.Is(err, domain.ErrInvalidArgument) ||
		errors.Is(err, domain.ErrNotFound) ||
		errors.Is(err, domain.ErrFailedPrecondition) ||
		errors.Is(err, domain.ErrForbidden)
}

// ProcessCommandMessage разбирает payload, ретраит транзиентные ошибки in-process
// и пишет в DLQ яд / исчерпанные ретраи. nil = offset можно коммитить.
func ProcessCommandMessage(ctx context.Context, raw []byte, handler OrderCommandHandler, dlq DeadLetterSink, maxAttempts int) error {
	if maxAttempts <= 0 {
		maxAttempts = defaultCommandMaxAttempts
	}
	cmd, err := ParseOrderCommand(raw)
	if err != nil {
		return publishCommandDLQ(ctx, dlq, raw, err)
	}
	if handler == nil {
		return publishCommandDLQ(ctx, dlq, raw, fmt.Errorf("%w: order command handler is not configured", domain.ErrFailedPrecondition))
	}

	var last error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		last = handler.Handle(ctx, cmd)
		if last == nil {
			return nil
		}
		if errors.Is(last, context.Canceled) || errors.Is(last, context.DeadlineExceeded) {
			return last
		}
		if IsPermanentCommandError(last) {
			return publishCommandDLQ(ctx, dlq, raw, last)
		}
	}
	return publishCommandDLQ(ctx, dlq, raw, last)
}

func publishCommandDLQ(ctx context.Context, dlq DeadLetterSink, original []byte, reason error) error {
	if dlq == nil {
		return fmt.Errorf("dlq: %w", reason)
	}
	msg := "unknown"
	if reason != nil {
		msg = reason.Error()
	}
	if err := dlq.Publish(ctx, original, msg); err != nil {
		return fmt.Errorf("dlq publish: %w", err)
	}
	return nil
}

type nopInbox struct{}

func (nopInbox) InsertIfNew(context.Context, string) (bool, error) {
	return true, nil
}

var (
	_ InboxStore          = nopInbox{}
	_ OrderCommandHandler = (*ApplyOrderCommand)(nil)
)
