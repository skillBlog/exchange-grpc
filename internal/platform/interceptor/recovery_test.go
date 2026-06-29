package interceptor_test

import (
	"context"
	"testing"

	"github.com/exchange-grpc/internal/platform/interceptor"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryServerPanicRecovery_returnsInternal(t *testing.T) {
	core, recorded := observer.New(zap.ErrorLevel)
	log := zap.New(core)

	handler := func(ctx context.Context, _ any) (any, error) {
		panic("boom")
	}

	_, err := interceptor.UnaryServerPanicRecovery(log)(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, handler)
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want Internal", status.Code(err))
	}

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1", len(entries))
	}
	if entries[0].Message != "grpc panic recovered" {
		t.Fatalf("log message = %q", entries[0].Message)
	}
}

func TestUnaryServerPanicRecovery_passesThroughSuccess(t *testing.T) {
	log := zap.NewNop()
	handler := func(ctx context.Context, _ any) (any, error) {
		return "ok", nil
	}

	resp, err := interceptor.UnaryServerPanicRecovery(log)(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if resp != "ok" {
		t.Fatalf("resp = %v, want ok", resp)
	}
}
