package mapper_test

import (
	"testing"
	"time"

	"github.com/exchange-grpc/internal/adapter/grpc/mapper"
	"github.com/exchange-grpc/internal/domain"
)

func TestMarketToProto(t *testing.T) {
	deletedAt := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)
	market := domain.Market{
		ID:           "BTC-USDT",
		Name:         "Bitcoin / Tether",
		BaseAsset:    "BTC",
		QuoteAsset:   "USDT",
		Enabled:      true,
		DeletedAt:    &deletedAt,
		AllowedRoles: []string{"trader"},
	}

	protoMarket := mapper.MarketToProto(market)

	if protoMarket.GetId() != market.ID {
		t.Fatalf("Id = %q, want %q", protoMarket.GetId(), market.ID)
	}
	if protoMarket.GetName() != market.Name {
		t.Fatalf("Name = %q, want %q", protoMarket.GetName(), market.Name)
	}
	if !protoMarket.GetEnabled() {
		t.Fatal("Enabled = false, want true")
	}
	if protoMarket.GetDeletedAt().AsTime() != deletedAt {
		t.Fatalf("DeletedAt = %v, want %v", protoMarket.GetDeletedAt().AsTime(), deletedAt)
	}
	if len(protoMarket.GetAllowedRoles()) != 1 || protoMarket.GetAllowedRoles()[0] != "trader" {
		t.Fatalf("AllowedRoles = %v", protoMarket.GetAllowedRoles())
	}
}

func TestMarketsToProto(t *testing.T) {
	markets := []domain.Market{
		{ID: "BTC-USDT", Enabled: true},
		{ID: "ETH-USDT", Enabled: true},
	}

	protoMarkets := mapper.MarketsToProto(markets)
	if len(protoMarkets) != 2 {
		t.Fatalf("len = %d, want 2", len(protoMarkets))
	}
	if protoMarkets[0].GetId() != "BTC-USDT" {
		t.Fatalf("first id = %q", protoMarkets[0].GetId())
	}
}

func TestMarketsToProto_emptySlice(t *testing.T) {
	protoMarkets := mapper.MarketsToProto(nil)
	if len(protoMarkets) != 0 {
		t.Fatalf("len = %d, want 0", len(protoMarkets))
	}
}

func TestMarketToProto_nilDeletedAt(t *testing.T) {
	protoMarket := mapper.MarketToProto(domain.Market{ID: "BTC-USDT", Enabled: true})
	if protoMarket.GetDeletedAt() != nil {
		t.Fatal("DeletedAt should be nil")
	}
}
