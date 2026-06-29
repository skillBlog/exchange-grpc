package interceptor

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryServerPanicRecovery преобразует панику в gRPC-ошибку Internal.
func UnaryServerPanicRecovery(log *zap.Logger) grpc.UnaryServerInterceptor {
	if log == nil {
		log = zap.NewNop()
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Error(
					"grpc panic recovered",
					zap.String("method", info.FullMethod),
					zap.String("request_id", RequestIDFromContext(ctx)),
					zap.Any("panic", recovered),
				)
				err = status.Error(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}
