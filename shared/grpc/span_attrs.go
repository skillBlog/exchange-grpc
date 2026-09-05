package grpc

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

// UnaryServerSpanAttrs добавляет безопасные атрибуты на RPC-спан (без email/credentials).
func UnaryServerSpanAttrs() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		setSafeSpanAttrs(ctx)
		return handler(ctx, req)
	}
}

// StreamServerSpanAttrs добавляет request_id на stream RPC-спан.
// user_id появляется в контексте только после JWT RecvMsg — его ставят бизнес-спаны.
func StreamServerSpanAttrs() grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		setSafeSpanAttrs(stream.Context())
		return handler(srv, stream)
	}
}

func setSafeSpanAttrs(ctx context.Context) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	attrs := make([]attribute.KeyValue, 0, 2)
	if id := RequestIDFromContext(ctx); id != "" {
		attrs = append(attrs, attribute.String("request_id", id))
	}
	if uid, ok := UserIDFromContext(ctx); ok {
		attrs = append(attrs, attribute.String("user_id", uid))
	}
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
}
