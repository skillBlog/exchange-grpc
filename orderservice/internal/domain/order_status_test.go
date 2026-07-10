package domain_test

import (
	"testing"

	"github.com/exchange-grpc/orderservice/internal/domain"
)

func TestValidateTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    domain.OrderStatus
		to      domain.OrderStatus
		wantErr bool
	}{
		{name: "created to filled", from: domain.OrderStatusCreated, to: domain.OrderStatusFilled},
		{name: "created to failed", from: domain.OrderStatusCreated, to: domain.OrderStatusFailed},
		{name: "failed to created", from: domain.OrderStatusFailed, to: domain.OrderStatusCreated, wantErr: true},
		{name: "filled to created", from: domain.OrderStatusFilled, to: domain.OrderStatusCreated, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := domain.ValidateTransition(tc.from, tc.to)
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
