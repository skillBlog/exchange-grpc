package grpc_test

import (
	"context"
	"errors"
	"testing"

	sharedgrpc "github.com/exchange-grpc/shared/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryServerRecovery_convertsPanicToInternal(t *testing.T) {
	interceptor := sharedgrpc.UnaryServerRecovery(nil)
	_, err := interceptor(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Panic"},
		func(context.Context, any) (any, error) {
			panic("boom")
		},
	)
	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %v, want Internal", status.Code(err))
	}
	if status.Convert(err).Message() != "internal error" {
		t.Fatalf("message = %q, want internal error", status.Convert(err).Message())
	}
}

func TestUnaryServerRecovery_passesThroughHandlerError(t *testing.T) {
	want := status.Error(codes.InvalidArgument, "bad request")
	interceptor := sharedgrpc.UnaryServerRecovery(nil)
	_, err := interceptor(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Ok"},
		func(context.Context, any) (any, error) {
			return nil, want
		},
	)
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want original handler error", err)
	}
}

type panicStream struct {
	grpc.ServerStream
}

func (panicStream) Context() context.Context { return context.Background() }

func TestStreamServerRecovery_convertsPanicToInternal(t *testing.T) {
	interceptor := sharedgrpc.StreamServerRecovery(nil)
	err := interceptor(
		nil,
		panicStream{},
		&grpc.StreamServerInfo{FullMethod: "/test.Service/StreamPanic"},
		func(any, grpc.ServerStream) error {
			panic("stream boom")
		},
	)
	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %v, want Internal", status.Code(err))
	}
}
