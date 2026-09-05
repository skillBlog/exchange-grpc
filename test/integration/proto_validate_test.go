package integration_test

import (
	"testing"

	"buf.build/go/protovalidate"
	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
	orderv1 "github.com/exchange-grpc/proto/pb/order/v1"
	spotv1 "github.com/exchange-grpc/proto/pb/spot/v1"
	"google.golang.org/protobuf/proto"
)

func TestProtoValidate_wave1Contracts(t *testing.T) {
	validator, err := protovalidate.New()
	if err != nil {
		t.Fatalf("protovalidate.New() error = %v", err)
	}

	validOrder := &orderv1.CreateOrderRequest{
		MarketId: "BTC-USDT",
		Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
		Quantity: &commonv1.Decimal{Value: "0.01"},
	}
	validMarket := &commonv1.Market{
		Id:         "BTC-USDT",
		Name:       "Bitcoin",
		BaseAsset:  "BTC",
		QuoteAsset: "USDT",
	}

	tests := []struct {
		name    string
		msg     proto.Message
		wantErr bool
	}{
		{name: "create order valid without price", msg: validOrder},
		{
			name: "create order valid with price",
			msg: &orderv1.CreateOrderRequest{
				MarketId: "BTC-USDT",
				Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
				Price:    &commonv1.Money{Amount: "100.5", Currency: "USD"},
				Quantity: &commonv1.Decimal{Value: "1"},
			},
		},
		{
			name: "quantity required",
			msg: &orderv1.CreateOrderRequest{
				MarketId: "BTC-USDT",
				Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
			},
			wantErr: true,
		},
		{
			name: "quantity zero rejected",
			msg: &orderv1.CreateOrderRequest{
				MarketId: "BTC-USDT",
				Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
				Quantity: &commonv1.Decimal{Value: "0"},
			},
			wantErr: true,
		},
		{
			name: "quantity 0.00 rejected",
			msg: &orderv1.CreateOrderRequest{
				MarketId: "BTC-USDT",
				Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
				Quantity: &commonv1.Decimal{Value: "0.00"},
			},
			wantErr: true,
		},
		{
			name: "price zero rejected when set",
			msg: &orderv1.CreateOrderRequest{
				MarketId: "BTC-USDT",
				Side:     commonv1.OrderSide_ORDER_SIDE_BUY,
				Price:    &commonv1.Money{Amount: "0", Currency: "USD"},
				Quantity: &commonv1.Decimal{Value: "1"},
			},
			wantErr: true,
		},
		{name: "market valid", msg: validMarket},
		{
			name: "market id not a symbol",
			msg: &commonv1.Market{
				Id:         "not-a-symbol",
				Name:       "Bitcoin",
				BaseAsset:  "BTC",
				QuoteAsset: "USDT",
			},
			wantErr: true,
		},
		{
			name: "base asset lowercase rejected",
			msg: &commonv1.Market{
				Id:         "BTC-USDT",
				Name:       "Bitcoin",
				BaseAsset:  "btc",
				QuoteAsset: "USDT",
			},
			wantErr: true,
		},
		{
			name: "get market id aligned with market.id",
			msg:  &spotv1.GetMarketRequest{MarketId: "ETH-USDT"},
		},
		{
			name:    "get market id lowercase rejected",
			msg:     &spotv1.GetMarketRequest{MarketId: "eth-usdt"},
			wantErr: true,
		},
		{name: "list orders empty filters valid", msg: &orderv1.ListOrdersRequest{}},
		{
			name: "list orders valid market filter",
			msg:  &orderv1.ListOrdersRequest{MarketId: proto.String("BTC-USDT")},
		},
		{
			name:    "list orders invalid market filter",
			msg:     &orderv1.ListOrdersRequest{MarketId: proto.String("btc-usdt")},
			wantErr: true,
		},
		{
			name: "list orders valid status filter",
			msg:  &orderv1.ListOrdersRequest{Status: commonv1.OrderStatus_ORDER_STATUS_CREATED.Enum()},
		},
		{
			name: "market valid with limits",
			msg: &commonv1.Market{
				Id:                "BTC-USDT",
				Name:              "Bitcoin",
				BaseAsset:         "BTC",
				QuoteAsset:        "USDT",
				MinOrderSize:      &commonv1.Decimal{Value: "0.0001"},
				QuantityPrecision: 8,
				MinNotional:       &commonv1.Decimal{Value: "10"},
			},
		},
		{
			name: "market min_order_size zero rejected",
			msg: &commonv1.Market{
				Id:           "BTC-USDT",
				Name:         "Bitcoin",
				BaseAsset:    "BTC",
				QuoteAsset:   "USDT",
				MinOrderSize: &commonv1.Decimal{Value: "0"},
			},
			wantErr: true,
		},
		{name: "stream user updates empty valid", msg: &orderv1.StreamUserOrderUpdatesRequest{}},
		{
			name: "stream user updates valid market filter",
			msg:  &orderv1.StreamUserOrderUpdatesRequest{MarketId: proto.String("BTC-USDT")},
		},
		{
			name:    "stream user updates invalid market filter",
			msg:     &orderv1.StreamUserOrderUpdatesRequest{MarketId: proto.String("btc-usdt")},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.Validate(tc.msg)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected validation error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}
