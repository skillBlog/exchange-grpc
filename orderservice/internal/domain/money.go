package domain

import (
	"fmt"
	"regexp"
	"strings"
)

// decimalPattern — неотрицательное десятичное число без экспоненты (например 10, 10.5, 0.01).
var decimalPattern = regexp.MustCompile(`^(?:0|[1-9]\d*)(?:\.\d+)?$`)

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
	if err := validateDecimalString(amount, "amount"); err != nil {
		return Money{}, err
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

// NewDecimal создаёт Decimal с валидацией числового формата.
func NewDecimal(value string) (Decimal, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Decimal{}, fmt.Errorf("%w: quantity is required", ErrInvalidArgument)
	}
	if err := validateDecimalString(value, "quantity"); err != nil {
		return Decimal{}, err
	}
	return Decimal{Value: value}, nil
}

func validateDecimalString(value, field string) error {
	if !decimalPattern.MatchString(value) {
		return fmt.Errorf("%w: invalid %s %q", ErrInvalidArgument, field, value)
	}
	return nil
}
