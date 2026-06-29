package interceptor_test

import (
	"context"
	"testing"

	"github.com/exchange-grpc/internal/platform/interceptor"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestUnaryServerRequestID_fromMetadata(t *testing.T) {
	want := uuid.NewString()
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(interceptor.MetadataKey, want))

	called := false
	handler := func(ctx context.Context, _ any) (any, error) {
		called = true
		if got := interceptor.RequestIDFromContext(ctx); got != want {
			t.Fatalf("RequestIDFromContext() = %q, want %q", got, want)
		}
		return nil, nil
	}

	_, err := interceptor.UnaryServerRequestID(ctx, nil, &grpc.UnaryServerInfo{}, handler)
	if err != nil {
		t.Fatalf("UnaryServerRequestID() error = %v", err)
	}
	if !called {
		t.Fatal("handler was not called")
	}
}

func TestUnaryServerRequestID_generatesWhenMissing(t *testing.T) {
	var got string
	handler := func(ctx context.Context, _ any) (any, error) {
		got = interceptor.RequestIDFromContext(ctx)
		return nil, nil
	}

	_, err := interceptor.UnaryServerRequestID(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	if err != nil {
		t.Fatalf("UnaryServerRequestID() error = %v", err)
	}
	if got == "" {
		t.Fatal("expected generated request id")
	}
	if _, err := uuid.Parse(got); err != nil {
		t.Fatalf("generated request id is not a valid UUID: %v", err)
	}
}

func TestUnaryClientRequestID_propagatesMetadata(t *testing.T) {
	want := uuid.NewString()
	ctx := interceptor.ContextWithRequestID(context.Background(), want)

	var outgoing metadata.MD
	invoker := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			t.Fatal("expected outgoing metadata")
		}
		outgoing = md
		return nil
	}

	if err := interceptor.UnaryClientRequestID(ctx, "/test.Service/Method", nil, nil, nil, invoker); err != nil {
		t.Fatalf("UnaryClientRequestID() error = %v", err)
	}

	values := outgoing.Get(interceptor.MetadataKey)
	if len(values) != 1 || values[0] != want {
		t.Fatalf("outgoing metadata = %v, want %q", values, want)
	}
}
