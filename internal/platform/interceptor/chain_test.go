package interceptor_test

import (
	"context"
	"testing"

	"github.com/exchange-grpc/internal/platform/interceptor"
	"google.golang.org/grpc"
)

func TestChainUnaryServer_order(t *testing.T) {
	var order []int

	mk := func(id int) grpc.UnaryServerInterceptor {
		return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			order = append(order, id)
			return handler(ctx, req)
		}
	}

	chain := interceptor.ChainUnaryServer(mk(1), mk(2), mk(3))
	handler := func(ctx context.Context, _ any) (any, error) {
		order = append(order, 4)
		return nil, nil
	}

	if _, err := chain(context.Background(), nil, &grpc.UnaryServerInfo{}, handler); err != nil {
		t.Fatalf("chain error = %v", err)
	}

	want := []int{1, 2, 3, 4}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
}
