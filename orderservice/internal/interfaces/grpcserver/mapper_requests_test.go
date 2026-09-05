package grpcserver

import (
	"testing"

	"github.com/exchange-grpc/orderservice/internal/domain"
	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
	orderv1 "github.com/exchange-grpc/proto/pb/order/v1"
	"google.golang.org/protobuf/proto"
)

func TestListOrdersRequestToInput_mapsOptionalFilters(t *testing.T) {
	input, err := Mapper{}.ListOrdersRequestToInput(&orderv1.ListOrdersRequest{
		MarketId: proto.String("BTC-USDT"),
		Status:   commonv1.OrderStatus_ORDER_STATUS_FILLED.Enum(),
	}, "11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("ListOrdersRequestToInput() error = %v", err)
	}
	if input.MarketID != "BTC-USDT" {
		t.Fatalf("MarketID = %q", input.MarketID)
	}
	if input.Status != domain.OrderStatusFilled {
		t.Fatalf("Status = %q, want filled", input.Status)
	}
}

func TestStreamUserOrderUpdatesRequestToInput_mapsOptionalFilters(t *testing.T) {
	input, err := Mapper{}.StreamUserOrderUpdatesRequestToInput(&orderv1.StreamUserOrderUpdatesRequest{
		MarketId: proto.String("ETH-USDT"),
		Status:   commonv1.OrderStatus_ORDER_STATUS_CREATED.Enum(),
	}, "user")
	if err != nil {
		t.Fatalf("StreamUserOrderUpdatesRequestToInput() error = %v", err)
	}
	if input.MarketID != "ETH-USDT" {
		t.Fatalf("MarketID = %q", input.MarketID)
	}
	if input.Status != domain.OrderStatusCreated {
		t.Fatalf("Status = %q, want created", input.Status)
	}
}

func TestListOrdersRequestToInput_emptyFilters(t *testing.T) {
	input, err := Mapper{}.ListOrdersRequestToInput(&orderv1.ListOrdersRequest{}, "user")
	if err != nil {
		t.Fatalf("ListOrdersRequestToInput() error = %v", err)
	}
	if input.MarketID != "" {
		t.Fatalf("MarketID = %q, want empty", input.MarketID)
	}
	if input.Status != "" {
		t.Fatalf("Status = %q, want empty", input.Status)
	}
}
