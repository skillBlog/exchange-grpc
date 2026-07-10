package domain

import (
	"fmt"
	"strings"
)

// Money — денежная сумма в USD.
type Money struct {
	Amount   string
	Currency string
}

// NewMoney создаёт Money из amount и currency.
func NewMoney(amount, currency string) (Money, error) {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return Money{}, nil
	}

	currency = strings.TrimSpace(strings.ToUpper(currency))
	if currency == "" {
		currency = "USD"
	}
	if currency != "USD" {
		return Money{}, fmt.Errorf("%w: only USD currency is supported", ErrInvalidArgument)
	}
	return Money{Amount: amount, Currency: currency}, nil
}

// IsZero сообщает, что цена не задана.
func (m Money) IsZero() bool {
	return strings.TrimSpace(m.Amount) == ""
}

// Decimal — количество актива.
type Decimal struct {
	Value string
}

// NewDecimal создаёт Decimal с валидацией.
func NewDecimal(value string) (Decimal, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Decimal{}, fmt.Errorf("%w: quantity is required", ErrInvalidArgument)
	}
	return Decimal{Value: value}, nil
}
