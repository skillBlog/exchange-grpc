package interceptor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/exchange-grpc/internal/platform/interceptor"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryServerLogger_includesRequestID(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	log := zap.New(core)

	const requestID = "req-123"
	ctx := interceptor.ContextWithRequestID(context.Background(), requestID)

	handler := func(ctx context.Context, _ any) (any, error) {
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/exchange.v1.OrderService/CreateOrder"}
	_, err := interceptor.UnaryServerLogger(log)(ctx, nil, info, handler)
	if err != nil {
		t.Fatalf("UnaryServerLogger() error = %v", err)
	}

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1", len(entries))
	}

	fields := entries[0].ContextMap()
	if fields["request_id"] != requestID {
		t.Fatalf("request_id = %v, want %q", fields["request_id"], requestID)
	}
	if fields["method"] != info.FullMethod {
		t.Fatalf("method = %v, want %q", fields["method"], info.FullMethod)
	}
}

func TestUnaryServerLogger_logsErrorCode(t *testing.T) {
	core, recorded := observer.New(zap.WarnLevel)
	log := zap.New(core)

	handlerErr := status.Error(codes.InvalidArgument, "bad input")
	handler := func(ctx context.Context, _ any) (any, error) {
		return nil, handlerErr
	}

	_, err := interceptor.UnaryServerLogger(log)(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, handler)
	if !errors.Is(err, handlerErr) {
		t.Fatalf("error = %v, want %v", err, handlerErr)
	}

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1", len(entries))
	}
	if entries[0].ContextMap()["grpc_code"] != codes.InvalidArgument.String() {
		t.Fatalf("grpc_code = %v, want %q", entries[0].ContextMap()["grpc_code"], codes.InvalidArgument.String())
	}
}
