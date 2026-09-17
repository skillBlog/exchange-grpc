package grpc_test

import (
	"context"
	"testing"

	sharedgrpc "github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/shared/sessionvalidation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type recvStream struct {
	grpc.ServerStream
}

func (recvStream) Context() context.Context { return context.Background() }

func (recvStream) RecvMsg(any) error { return nil }

type countingStream struct {
	grpc.ServerStream
	ctx       context.Context
	recvCount int
}

func (s *countingStream) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}

func (s *countingStream) RecvMsg(any) error {
	s.recvCount++
	return nil
}

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

func TestStreamServerJWTAuth_protectedRequiresAuthBeforeHandler(t *testing.T) {
	tokens, err := sessionvalidation.NewTokenService("stream-jwt-test", 0)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	stream := &countingStream{}
	called := false
	interceptor := sharedgrpc.NewStreamServerJWTAuth(tokens)
	err = interceptor(
		nil,
		stream,
		&grpc.StreamServerInfo{FullMethod: "/order.v1.OrderService/StreamOrderUpdates"},
		func(_ any, _ grpc.ServerStream) error {
			called = true
			return nil
		},
	)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
	}
	if called {
		t.Fatal("handler must not run without JWT")
	}
	if stream.recvCount != 0 {
		t.Fatalf("RecvMsg called %d times, want 0", stream.recvCount)
	}
}

func TestStreamServerJWTAuth_enrichesContextWithoutRecv(t *testing.T) {
	tokens, err := sessionvalidation.NewTokenService("stream-jwt-test", 0)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	raw, err := tokens.Issue("user-1", nil)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"authorization", "Bearer "+raw,
	))
	stream := &countingStream{ctx: ctx}
	interceptor := sharedgrpc.NewStreamServerJWTAuth(tokens)
	var gotUser string
	err = interceptor(
		nil,
		stream,
		&grpc.StreamServerInfo{FullMethod: "/order.v1.OrderService/StreamOrderUpdates"},
		func(_ any, ss grpc.ServerStream) error {
			gotUser, _ = sharedgrpc.UserIDFromContext(ss.Context())
			return nil
		},
	)
	if err != nil {
		t.Fatalf("interceptor error = %v", err)
	}
	if gotUser != "user-1" {
		t.Fatalf("user_id = %q, want user-1", gotUser)
	}
	if stream.recvCount != 0 {
		t.Fatalf("RecvMsg called %d times, want 0", stream.recvCount)
	}
}

func TestUnaryServerJWTAuth_readsTrimmedBearer(t *testing.T) {
	tokens, err := sessionvalidation.NewTokenService("unary-jwt-test", 0)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	raw, err := tokens.Issue("user-1", nil)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"authorization", "  Bearer "+raw+"  ",
	))
	interceptor := sharedgrpc.NewUnaryServerJWTAuth(tokens)
	var gotUser string
	_, err = interceptor(
		ctx,
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"},
		func(ctx context.Context, _ any) (any, error) {
			gotUser, _ = sharedgrpc.UserIDFromContext(ctx)
			return "ok", nil
		},
	)
	if err != nil {
		t.Fatalf("interceptor error = %v", err)
	}
	if gotUser != "user-1" {
		t.Fatalf("user_id = %q, want user-1", gotUser)
	}
}

func TestUnaryServerJWTAuth_invalidTokenUnauthenticated(t *testing.T) {
	tokens, err := sessionvalidation.NewTokenService("unary-jwt-test", 0)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"authorization", "Bearer not-a-jwt",
	))
	interceptor := sharedgrpc.NewUnaryServerJWTAuth(tokens)
	_, err = interceptor(
		ctx,
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"},
		func(context.Context, any) (any, error) {
			t.Fatal("handler should not run")
			return nil, nil
		},
	)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
	}
}

func TestUnaryServerJWTAuth_whitespaceAuthorizationRejected(t *testing.T) {
	tokens, err := sessionvalidation.NewTokenService("unary-jwt-test", 0)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"authorization", "   ",
	))
	interceptor := sharedgrpc.NewUnaryServerJWTAuth(tokens)
	_, err = interceptor(
		ctx,
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"},
		func(context.Context, any) (any, error) {
			t.Fatal("handler should not run")
			return nil, nil
		},
	)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
	}
}
