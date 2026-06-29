package interceptor

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

// UnaryServerSpanRequestID добавляет x-request-id в атрибуты активного trace span.
func UnaryServerSpanRequestID(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		requestID := RequestIDFromContext(ctx)
		if requestID != "" {
			span.SetAttributes(attribute.String("request_id", requestID))
		}
		span.SetAttributes(attribute.String("rpc.method", info.FullMethod))
	}

	return handler(ctx, req)
}
