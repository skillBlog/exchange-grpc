package main

import (
	"testing"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
)

func TestParseOrderType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    exchangev1.OrderType
		wantErr bool
	}{
		{name: "limit", input: "limit", want: exchangev1.OrderType_ORDER_TYPE_LIMIT},
		{name: "market upper", input: "MARKET", want: exchangev1.OrderType_ORDER_TYPE_MARKET},
		{name: "invalid", input: "stop", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOrderType(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseOrderType() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("got = %v, want %v", got, tt.want)
			}
		})
	}
}
