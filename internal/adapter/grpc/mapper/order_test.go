package mapper_test

import (
	"testing"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	"github.com/exchange-grpc/internal/adapter/grpc/mapper"
	"github.com/exchange-grpc/internal/domain"
)

func TestOrderTypeToDomain(t *testing.T) {
	tests := []struct {
		name      string
		protoType exchangev1.OrderType
		want      domain.OrderType
		wantErr   bool
	}{
		{name: "limit", protoType: exchangev1.OrderType_ORDER_TYPE_LIMIT, want: domain.OrderTypeLimit},
		{name: "market", protoType: exchangev1.OrderType_ORDER_TYPE_MARKET, want: domain.OrderTypeMarket},
		{name: "unspecified", protoType: exchangev1.OrderType_ORDER_TYPE_UNSPECIFIED, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapper.OrderTypeToDomain(tt.protoType)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("OrderTypeToDomain() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("got = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOrderToGetOrderStatusResponse(t *testing.T) {
	order := domain.Order{
		ID:       "order-1",
		UserID:   "user-1",
		MarketID: "BTC-USDT",
		Type:     domain.OrderTypeLimit,
		Price:    "100",
		Quantity: "0.1",
		Status:   domain.OrderStatusCreated,
	}

	resp := mapper.OrderToGetOrderStatusResponse(order)
	if resp.GetOrderId() != order.ID {
		t.Fatalf("OrderId = %q", resp.GetOrderId())
	}
	if resp.GetOrderType() != exchangev1.OrderType_ORDER_TYPE_LIMIT {
		t.Fatalf("OrderType = %v", resp.GetOrderType())
	}
	if resp.GetStatus() != string(domain.OrderStatusCreated) {
		t.Fatalf("Status = %q", resp.GetStatus())
	}
}
