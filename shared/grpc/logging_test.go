package grpc_test

import (
	"context"
	"errors"
	"testing"

	sharedgrpc "github.com/exchange-grpc/shared/grpc"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryServerLogging_errorLevels(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		level zapcore.Level
		msg   string
	}{
		{
			name:  "invalid argument is debug",
			err:   status.Error(codes.InvalidArgument, "bad request"),
			level: zapcore.DebugLevel,
			msg:   "grpc request failed",
		},
		{
			name:  "not found is debug",
			err:   status.Error(codes.NotFound, "missing"),
			level: zapcore.DebugLevel,
			msg:   "grpc request failed",
		},
		{
			name:  "permission denied is debug",
			err:   status.Error(codes.PermissionDenied, "no access"),
			level: zapcore.DebugLevel,
			msg:   "grpc request failed",
		},
		{
			name:  "unauthenticated is debug",
			err:   status.Error(codes.Unauthenticated, "no token"),
			level: zapcore.DebugLevel,
			msg:   "grpc request failed",
		},
		{
			name:  "resource exhausted is warn",
			err:   status.Error(codes.ResourceExhausted, "rate limited"),
			level: zapcore.WarnLevel,
			msg:   "grpc request failed",
		},
		{
			name:  "internal is error",
			err:   status.Error(codes.Internal, "db down"),
			level: zapcore.ErrorLevel,
			msg:   "grpc request failed",
		},
		{
			name:  "unavailable is error",
			err:   status.Error(codes.Unavailable, "dep down"),
			level: zapcore.ErrorLevel,
			msg:   "grpc request failed",
		},
		{
			name:  "plain error is error",
			err:   errors.New("not a grpc status"),
			level: zapcore.ErrorLevel,
			msg:   "grpc request failed",
		},
		{
			name:  "success is debug",
			err:   nil,
			level: zapcore.DebugLevel,
			msg:   "grpc request",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			log, logs := observedLogger(t)
			interceptor := sharedgrpc.UnaryServerLogging(log)
			_, err := interceptor(
				context.Background(),
				nil,
				&grpc.UnaryServerInfo{FullMethod: "/test.Service/Call"},
				func(context.Context, any) (any, error) {
					return "ok", tc.err
				},
			)
			if !errors.Is(err, tc.err) {
				t.Fatalf("returned err = %v, want %v", err, tc.err)
			}

			entries := logs.All()
			if len(entries) != 1 {
				t.Fatalf("log entries = %d, want 1", len(entries))
			}
			got := entries[0]
			if got.Level != tc.level {
				t.Fatalf("level = %s, want %s", got.Level, tc.level)
			}
			if got.Message != tc.msg {
				t.Fatalf("message = %q, want %q", got.Message, tc.msg)
			}
		})
	}
}

func TestStreamServerLogging_errorLevels(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		level zapcore.Level
		msg   string
	}{
		{
			name:  "not found is debug",
			err:   status.Error(codes.NotFound, "missing"),
			level: zapcore.DebugLevel,
			msg:   "grpc stream failed",
		},
		{
			name:  "internal is error",
			err:   status.Error(codes.Internal, "db down"),
			level: zapcore.ErrorLevel,
			msg:   "grpc stream failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			log, logs := observedLogger(t)
			interceptor := sharedgrpc.StreamServerLogging(log)
			err := interceptor(
				nil,
				panicStream{},
				&grpc.StreamServerInfo{FullMethod: "/test.Service/Stream"},
				func(any, grpc.ServerStream) error {
					return tc.err
				},
			)
			if !errors.Is(err, tc.err) {
				t.Fatalf("returned err = %v, want %v", err, tc.err)
			}

			var failed observer.LoggedEntry
			found := false
			for _, entry := range logs.All() {
				if entry.Message == tc.msg {
					failed = entry
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("missing log %q in %+v", tc.msg, logs.All())
			}
			if failed.Level != tc.level {
				t.Fatalf("level = %s, want %s", failed.Level, tc.level)
			}
		})
	}
}

func observedLogger(t *testing.T) (*zap.Logger, *observer.ObservedLogs) {
	t.Helper()
	core, logs := observer.New(zapcore.DebugLevel)
	return zap.New(core), logs
}
