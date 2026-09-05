package domain_test

import (
	"errors"
	"testing"

	"github.com/exchange-grpc/orderservice/internal/domain"
)

func TestValidateOrderAgainstMarket(t *testing.T) {
	qty := func(v string) domain.Decimal {
		t.Helper()
		d, err := domain.NewDecimal(v)
		if err != nil {
			t.Fatalf("NewDecimal(%q) error = %v", v, err)
		}
		return d
	}
	price := func(v string) domain.Money {
		t.Helper()
		m, err := domain.NewMoney(v, "USD")
		if err != nil {
			t.Fatalf("NewMoney(%q) error = %v", v, err)
		}
		return m
	}

	btc := domain.MarketLimits{MinOrderSize: "0.0001", QuantityPrecision: 8, MinNotional: "10"}

	tests := []struct {
		name    string
		qty     domain.Decimal
		price   domain.Money
		limits  domain.MarketLimits
		wantErr error
	}{
		{name: "empty limits skip", qty: qty("0.1"), limits: domain.MarketLimits{}},
		{name: "equal to min size", qty: qty("0.0001"), limits: btc},
		{name: "above min size", qty: qty("0.1"), limits: btc},
		{
			name:    "below min size",
			qty:     qty("0.00001"),
			limits:  btc,
			wantErr: domain.ErrInvalidArgument,
		},
		{name: "precision at limit", qty: qty("0.00010001"), limits: btc},
		{
			name:    "precision exceeded",
			qty:     qty("0.000100001"),
			limits:  btc,
			wantErr: domain.ErrInvalidArgument,
		},
		{name: "trailing zeros do not count", qty: qty("0.10"), limits: domain.MarketLimits{QuantityPrecision: 1}},
		{name: "market order skips notional", qty: qty("0.1"), limits: btc},
		{
			name:    "notional below min",
			qty:     qty("0.01"),
			price:   price("100"),
			limits:  btc,
			wantErr: domain.ErrInvalidArgument,
		},
		{name: "notional equal min", qty: qty("0.1"), price: price("100"), limits: btc},
		{
			name:    "invalid configured min size",
			qty:     qty("0.1"),
			limits:  domain.MarketLimits{MinOrderSize: "abc"},
			wantErr: domain.ErrFailedPrecondition,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := domain.ValidateOrderAgainstMarket(tc.qty, tc.price, tc.limits)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
