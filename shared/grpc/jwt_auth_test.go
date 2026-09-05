package grpc_test

import (
	"context"
	"testing"

	sharedgrpc "github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/shared/sessionvalidation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type recvStream struct {
	grpc.ServerStream
}

func (recvStream) Context() context.Context { return context.Background() }

func (recvStream) RecvMsg(any) error { return nil }

func TestStreamServerJWTAuth_publicWatchSkipsAuth(t *testing.T) {
	interceptor := sharedgrpc.NewStreamServerJWTAuth(nil, grpc_health_v1.Health_Watch_FullMethodName)
	called := false
	err := interceptor(
		nil,
		recvStream{},
		&grpc.StreamServerInfo{FullMethod: grpc_health_v1.Health_Watch_FullMethodName},
		func(any, grpc.ServerStream) error {
			called = true
			return nil
		},
	)
	if err != nil {
		t.Fatalf("Watch error = %v", err)
	}
	if !called {
		t.Fatal("expected handler to run without JWT")
	}
}

func TestStreamServerJWTAuth_protectedRequiresAuthOnRecv(t *testing.T) {
	tokens, err := sessionvalidation.NewTokenService("stream-jwt-test", 0)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	interceptor := sharedgrpc.NewStreamServerJWTAuth(tokens)
	err = interceptor(
		nil,
		recvStream{},
		&grpc.StreamServerInfo{FullMethod: "/order.v1.OrderService/StreamOrderUpdates"},
		func(_ any, stream grpc.ServerStream) error {
			return stream.RecvMsg(nil)
		},
	)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
	}
}
