package grpc

import (
	"context"
	"runtime/debug"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const recoveredInternalMessage = "internal error"

// UnaryServerRecovery перехватывает panic в unary RPC и возвращает codes.Internal.
func UnaryServerRecovery(log *zap.Logger) grpc.UnaryServerInterceptor {
	if log == nil {
		log = zap.NewNop()
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Error("grpc unary panic recovered",
					zap.String("method", info.FullMethod),
					zap.String("request_id", RequestIDFromContext(ctx)),
					zap.Any("panic", recovered),
					zap.ByteString("stack", debug.Stack()),
				)
				resp = nil
				err = status.Error(codes.Internal, recoveredInternalMessage)
			}
		}()
		return handler(ctx, req)
	}
}

// StreamServerRecovery перехватывает panic в streaming RPC и возвращает codes.Internal.
func StreamServerRecovery(log *zap.Logger) grpc.StreamServerInterceptor {
	if log == nil {
		log = zap.NewNop()
	}

	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Error("grpc stream panic recovered",
					zap.String("method", info.FullMethod),
					zap.String("request_id", RequestIDFromContext(stream.Context())),
					zap.Any("panic", recovered),
					zap.ByteString("stack", debug.Stack()),
				)
				err = status.Error(codes.Internal, recoveredInternalMessage)
			}
		}()
		return handler(srv, stream)
	}
}
