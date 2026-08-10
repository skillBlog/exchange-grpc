package domain_test

import (
	"errors"
	"testing"

	"github.com/exchange-grpc/orderservice/internal/domain"
)

func TestNewDecimal(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "integer", value: "10"},
		{name: "fraction", value: "10.5"},
		{name: "leading zero fraction", value: "0.01"},
		{name: "zero", value: "0"},
		{name: "empty", value: "", wantErr: true},
		{name: "spaces only", value: "   ", wantErr: true},
		{name: "letters", value: "abc", wantErr: true},
		{name: "comma decimal", value: "10,5", wantErr: true},
		{name: "negative", value: "-1", wantErr: true},
		{name: "leading zeros", value: "01", wantErr: true},
		{name: "trailing dot", value: "10.", wantErr: true},
		{name: "scientific", value: "1e2", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := domain.NewDecimal(tc.value)
			if tc.wantErr {
				if !errors.Is(err, domain.ErrInvalidArgument) {
					t.Fatalf("error = %v, want ErrInvalidArgument", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Value != tc.value {
				t.Fatalf("value = %q, want %q", got.Value, tc.value)
			}
		})
	}
}

func TestNewMoney_amountFormat(t *testing.T) {
	if _, err := domain.NewMoney("10.5", "USD"); err != nil {
		t.Fatalf("valid amount error = %v", err)
	}
	if _, err := domain.NewMoney("abc", "USD"); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("invalid amount error = %v, want ErrInvalidArgument", err)
	}
	if _, err := domain.NewMoney("", "USD"); err != nil {
		t.Fatalf("empty amount should be zero money, got %v", err)
	}
}

func TestParseUUID(t *testing.T) {
	const valid = "550e8400-e29b-41d4-a716-446655440000"

	got, err := domain.ParseUUID(valid, "order_id")
	if err != nil {
		t.Fatalf("ParseUUID() error = %v", err)
	}
	if got != valid {
		t.Fatalf("got %q, want %q", got, valid)
	}

	if _, err := domain.ParseUUID("not-a-uuid", "user_id"); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
	if _, err := domain.ParseUUID("", "user_id"); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("empty error = %v, want ErrInvalidArgument", err)
	}
}
