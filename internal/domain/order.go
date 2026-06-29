package domain

import (
	"fmt"
	"strings"
)

// OrderType описывает способ исполнения ордера.
type OrderType string

const (
	OrderTypeLimit  OrderType = "limit"
	OrderTypeMarket OrderType = "market"
)

// OrderStatus описывает состояние ордера в жизненном цикле.
type OrderStatus string

const (
	OrderStatusCreated OrderStatus = "created"
	OrderStatusFilled  OrderStatus = "filled"
)

// Order — заявка пользователя на сделку на спотовом рынке.
type Order struct {
	ID       string
	UserID   string
	MarketID string
	Type     OrderType
	Price    string
	Quantity string
	Status   OrderStatus
}

// NewOrder создаёт ордер с валидацией полей и начальным статусом.
func NewOrder(id, userID, marketID string, orderType OrderType, price, quantity string) (Order, error) {
	if err := validateOrderInput(userID, marketID, orderType, price, quantity); err != nil {
		return Order{}, err
	}

	return Order{
		ID:       id,
		UserID:   userID,
		MarketID: marketID,
		Type:     orderType,
		Price:    price,
		Quantity: quantity,
		Status:   OrderStatusCreated,
	}, nil
}

func validateOrderInput(userID, marketID string, orderType OrderType, price, quantity string) error {
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("%w: user_id is required", ErrInvalidArgument)
	}
	if strings.TrimSpace(marketID) == "" {
		return fmt.Errorf("%w: market_id is required", ErrInvalidArgument)
	}
	if orderType != OrderTypeLimit && orderType != OrderTypeMarket {
		return fmt.Errorf("%w: unsupported order type %q", ErrInvalidArgument, orderType)
	}
	if strings.TrimSpace(quantity) == "" {
		return fmt.Errorf("%w: quantity is required", ErrInvalidArgument)
	}
	if orderType == OrderTypeLimit && strings.TrimSpace(price) == "" {
		return fmt.Errorf("%w: price is required for limit orders", ErrInvalidArgument)
	}
	return nil
}
