package domain_test

import (
	"testing"
	"time"

	"github.com/exchange-grpc/internal/domain"
)

func TestMarket_IsActive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		market  domain.Market
		want    bool
	}{
		{
			name:   "enabled without deleted_at",
			market: domain.Market{Enabled: true},
			want:   true,
		},
		{
			name:   "disabled",
			market: domain.Market{Enabled: false},
			want:   false,
		},
		{
			name:   "deleted",
			market: domain.Market{Enabled: true, DeletedAt: &now},
			want:   false,
		},
		{
			name:   "disabled and deleted",
			market: domain.Market{Enabled: false, DeletedAt: &now},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.market.IsActive(); got != tt.want {
				t.Fatalf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarket_IsAccessibleBy(t *testing.T) {
	market := domain.Market{
		AllowedRoles: []string{"trader", "admin"},
	}

	tests := []struct {
		name      string
		market    domain.Market
		userRoles []string
		want      bool
	}{
		{
			name:      "open market",
			market:    domain.Market{},
			userRoles: nil,
			want:      true,
		},
		{
			name:      "matching role",
			market:    market,
			userRoles: []string{"viewer", "trader"},
			want:      true,
		},
		{
			name:      "no matching role",
			market:    market,
			userRoles: []string{"viewer"},
			want:      false,
		},
		{
			name:      "empty user roles on restricted market",
			market:    market,
			userRoles: []string{},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.market.IsAccessibleBy(tt.userRoles); got != tt.want {
				t.Fatalf("IsAccessibleBy() = %v, want %v", got, tt.want)
			}
		})
	}
}
