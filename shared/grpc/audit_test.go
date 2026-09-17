package grpc_test

import (
	"context"
	"testing"

	sharedgrpc "github.com/exchange-grpc/shared/grpc"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogAudit_addsUserAndRequestID(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	ctx := sharedgrpc.ContextWithRequestID(context.Background(), "req-1")

	sharedgrpc.LogAudit(ctx, zap.New(core), "order created", "user-1", zap.String("order_id", "ord-1"))

	entries := logs.FilterMessage("order created").All()
	if len(entries) != 1 {
		t.Fatalf("logs = %d, want 1", len(entries))
	}
	fields := entries[0].ContextMap()
	if fields["user_id"] != "user-1" {
		t.Fatalf("user_id = %v", fields["user_id"])
	}
	if fields["request_id"] != "req-1" {
		t.Fatalf("request_id = %v", fields["request_id"])
	}
	if fields["order_id"] != "ord-1" {
		t.Fatalf("order_id = %v", fields["order_id"])
	}
}

func TestLogAudit_nilLogger(t *testing.T) {
	sharedgrpc.LogAudit(context.Background(), nil, "noop", "user-1")
}
