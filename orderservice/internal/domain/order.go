package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// OrderSide описывает направление сделки.
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// OrderStatus описывает состояние ордера в жизненном цикле.
type OrderStatus string

const (
	OrderStatusCreated   OrderStatus = "created"
	OrderStatusFilled    OrderStatus = "filled"
	OrderStatusRejected  OrderStatus = "rejected"
	OrderStatusFailed    OrderStatus = "failed"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// Order — market-заявка пользователя на спотовом рынке.
type Order struct {
	ID        string
	UserID    string
	MarketID  string
	Side      OrderSide
	Price     Money
	Quantity  Decimal
	Status    OrderStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewOrder создаёт market-ордер с валидацией полей и начальным статусом.
func NewOrder(id, userID, marketID string, side OrderSide, price Money, quantity Decimal, now time.Time) (Order, error) {
	if err := validateOrderInput(userID, marketID, side, quantity); err != nil {
		return Order{}, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	return Order{
		ID:        id,
		UserID:    userID,
		MarketID:  marketID,
		Side:      side,
		Price:     price,
		Quantity:  quantity,
		Status:    OrderStatusCreated,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// NewOrderID генерирует новый идентификатор ордера.
func NewOrderID() string {
	return uuid.NewString()
}

func validateOrderInput(userID, marketID string, side OrderSide, quantity Decimal) error {
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("%w: user_id is required", ErrInvalidArgument)
	}
	if strings.TrimSpace(marketID) == "" {
		return fmt.Errorf("%w: market_id is required", ErrInvalidArgument)
	}
	if side != OrderSideBuy && side != OrderSideSell {
		return fmt.Errorf("%w: unsupported order side %q", ErrInvalidArgument, side)
	}
	if strings.TrimSpace(quantity.Value) == "" {
		return fmt.Errorf("%w: quantity is required", ErrInvalidArgument)
	}
	return nil
}
