package main

import (
	"testing"

	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
)

func TestParseOrderSide(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    commonv1.OrderSide
		wantErr bool
	}{
		{name: "buy", input: "buy", want: commonv1.OrderSide_ORDER_SIDE_BUY},
		{name: "sell upper", input: "SELL", want: commonv1.OrderSide_ORDER_SIDE_SELL},
		{name: "invalid", input: "hold", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOrderSide(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseOrderSide() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("got = %v, want %v", got, tt.want)
			}
		})
	}
}
