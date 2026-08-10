package domain

import "fmt"

// CanTransitionTo проверяет допустимость перехода статуса ордера.
func (s OrderStatus) CanTransitionTo(next OrderStatus) bool {
	if s == next {
		return true
	}

	switch s {
	case OrderStatusCreated:
		return next == OrderStatusFilled ||
			next == OrderStatusRejected ||
			next == OrderStatusFailed ||
			next == OrderStatusCancelled
	case OrderStatusFilled, OrderStatusRejected, OrderStatusFailed, OrderStatusCancelled:
		return false
	default:
		return false
	}
}

// IsTerminal сообщает, что ордер больше не меняет статус.
func (s OrderStatus) IsTerminal() bool {
	switch s {
	case OrderStatusFilled, OrderStatusRejected, OrderStatusFailed, OrderStatusCancelled:
		return true
	default:
		return false
	}
}

// ValidateTransition возвращает ошибку при недопустимом переходе статуса.
func ValidateTransition(from, to OrderStatus) error {
	if from.CanTransitionTo(to) {
		return nil
	}
	return fmt.Errorf("%w: cannot transition from %q to %q", ErrInvalidArgument, from, to)
}
