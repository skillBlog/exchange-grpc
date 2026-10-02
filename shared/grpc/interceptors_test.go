package grpc_test

import (
	"context"
	"testing"

	sharedgrpc "github.com/exchange-grpc/shared/grpc"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestUnaryChain_requestIDBeforeRecovery_logsRequestID(t *testing.T) {
	log, logs := observedLogger(t)
	interceptor := sharedgrpc.ChainUnaryServer(
		sharedgrpc.UnaryServerRequestID,
		sharedgrpc.UnaryServerRecovery(log),
	)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		sharedgrpc.MetadataKey, "req-from-client",
	))
	_, err := interceptor(
		ctx,
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Panic"},
		func(context.Context, any) (any, error) {
			panic("boom")
		},
	)
	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %v, want Internal", status.Code(err))
	}

	entry := mustFindLog(t, logs, "grpc unary panic recovered")
	if got := logFieldString(t, entry, "request_id"); got != "req-from-client" {
		t.Fatalf("request_id = %q, want req-from-client", got)
	}
}

func TestStreamChain_requestIDBeforeRecovery_logsRequestID(t *testing.T) {
	log, logs := observedLogger(t)
	interceptor := sharedgrpc.ChainStreamServer(
		sharedgrpc.StreamServerRequestID,
		sharedgrpc.StreamServerRecovery(log),
	)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		sharedgrpc.MetadataKey, "stream-req-from-client",
	))
	err := interceptor(
		nil,
		ctxStream{ctx: ctx},
		&grpc.StreamServerInfo{FullMethod: "/test.Service/StreamPanic"},
		func(any, grpc.ServerStream) error {
			panic("stream boom")
		},
	)
	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %v, want Internal", status.Code(err))
	}

	entry := mustFindLog(t, logs, "grpc stream panic recovered")
	if got := logFieldString(t, entry, "request_id"); got != "stream-req-from-client" {
		t.Fatalf("request_id = %q, want stream-req-from-client", got)
	}
}

type ctxStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s ctxStream) Context() context.Context { return s.ctx }

func mustFindLog(t *testing.T, logs *observer.ObservedLogs, msg string) observer.LoggedEntry {
	t.Helper()
	for _, entry := range logs.All() {
		if entry.Message == msg {
			return entry
		}
	}
	t.Fatalf("missing log %q in %+v", msg, logs.All())
	return observer.LoggedEntry{}
}

func logFieldString(t *testing.T, entry observer.LoggedEntry, key string) string {
	t.Helper()
	for _, field := range entry.Context {
		if field.Key != key {
			continue
		}
		if field.Type == zapcore.StringType {
			return field.String
		}
		t.Fatalf("field %q type = %v, want string", key, field.Type)
	}
	t.Fatalf("missing field %q in %+v", key, entry.Context)
	return ""
}
