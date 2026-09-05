package domain

import (
	"fmt"
	"math/big"
	"strings"
)

// MarketLimits — торговые ограничения рынка. Пустое поле = правило не применяется.
type MarketLimits struct {
	MinOrderSize      string
	QuantityPrecision uint32
	MinNotional       string
}

// ValidateOrderAgainstMarket проверяет quantity и notional против лимитов рынка.
// min_notional считается только если задана цена (иначе notional неизвестен).
func ValidateOrderAgainstMarket(quantity Decimal, price Money, limits MarketLimits) error {
	if err := validateDecimalString(quantity.Value, "quantity"); err != nil {
		return err
	}

	qty, ok := parseDecimalRat(quantity.Value)
	if !ok {
		return fmt.Errorf("%w: invalid quantity %q", ErrInvalidArgument, quantity.Value)
	}

	if limits.MinOrderSize != "" {
		minSize, ok := parseDecimalRat(limits.MinOrderSize)
		if !ok {
			return fmt.Errorf("%w: market min_order_size is invalid", ErrFailedPrecondition)
		}
		if qty.Cmp(minSize) < 0 {
			return fmt.Errorf("%w: quantity is below market min_order_size", ErrInvalidArgument)
		}
	}

	if limits.QuantityPrecision > 0 {
		scale := fractionalScale(quantity.Value)
		if scale > int(limits.QuantityPrecision) {
			return fmt.Errorf("%w: quantity exceeds market quantity_precision", ErrInvalidArgument)
		}
	}

	if limits.MinNotional != "" && !price.IsZero() {
		minNotional, ok := parseDecimalRat(limits.MinNotional)
		if !ok {
			return fmt.Errorf("%w: market min_notional is invalid", ErrFailedPrecondition)
		}
		priceAmt, ok := parseDecimalRat(price.Amount)
		if !ok {
			return fmt.Errorf("%w: invalid price amount %q", ErrInvalidArgument, price.Amount)
		}
		notional := new(big.Rat).Mul(priceAmt, qty)
		if notional.Cmp(minNotional) < 0 {
			return fmt.Errorf("%w: order notional is below market min_notional", ErrInvalidArgument)
		}
	}

	return nil
}

func parseDecimalRat(value string) (*big.Rat, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, false
	}
	r, ok := new(big.Rat).SetString(value)
	return r, ok
}

func fractionalScale(value string) int {
	value = strings.TrimSpace(value)
	dot := strings.IndexByte(value, '.')
	if dot < 0 {
		return 0
	}
	return len(strings.TrimRight(value[dot+1:], "0"))
}
