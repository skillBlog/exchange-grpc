package application_test

import (
	"testing"

	"github.com/exchange-grpc/orderservice/internal/application"
)

func TestCreateOrderRateTierFromRoles(t *testing.T) {
	tests := []struct {
		name  string
		roles []string
		want  application.CreateOrderRateTier
	}{
		{name: "basic user", roles: []string{"user"}, want: application.CreateOrderRateTierBasic},
		{name: "premium trader", roles: []string{"Trader"}, want: application.CreateOrderRateTierPremium},
		{name: "admin wins", roles: []string{"trader", "Admin"}, want: application.CreateOrderRateTierAdmin},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := application.CreateOrderRateTierFromRoles(tc.roles)
			if got != tc.want {
				t.Fatalf("tier = %q, want %q", got, tc.want)
			}
		})
	}
}
