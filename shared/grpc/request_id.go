package grpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// MetadataKey — ключ gRPC metadata для корреляции запросов.
	MetadataKey = "x-request-id"
)

type requestIDContextKey struct{}

// RequestIDFromContext возвращает идентификатор запроса из контекста.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDContextKey{}).(string); ok {
		return id
	}
	return ""
}

// ContextWithRequestID возвращает дочерний контекст с заданным идентификатором запроса.
func ContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, id)
}

func requestIDFromMetadata(md metadata.MD) string {
	values := md.Get(MetadataKey)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// UnaryServerRequestID гарантирует наличие x-request-id в контексте каждого unary RPC.
func UnaryServerRequestID(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	requestID := RequestIDFromContext(ctx)
	if requestID == "" {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			requestID = requestIDFromMetadata(md)
		}
	}
	if requestID == "" {
		requestID = uuid.NewString()
	}

	ctx = ContextWithRequestID(ctx, requestID)
	return handler(ctx, req)
}

// UnaryClientRequestID пробрасывает x-request-id из контекста в исходящие metadata.
func UnaryClientRequestID(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	requestID := RequestIDFromContext(ctx)
	if requestID != "" {
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.MD{}
		} else {
			md = md.Copy()
		}
		md.Set(MetadataKey, requestID)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	return invoker(ctx, method, req, reply, cc, opts...)
}

// StreamServerRequestID гарантирует наличие x-request-id в контексте streaming RPC.
func StreamServerRequestID(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx := stream.Context()
	requestID := RequestIDFromContext(ctx)
	if requestID == "" {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			requestID = requestIDFromMetadata(md)
		}
	}
	if requestID == "" {
		requestID = uuid.NewString()
	}
	ctx = ContextWithRequestID(ctx, requestID)
	return handler(srv, &wrappedServerStream{ServerStream: stream, ctx: ctx})
}
