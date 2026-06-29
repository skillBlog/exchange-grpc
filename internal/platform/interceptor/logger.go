package interceptor

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryServerLogger логирует завершение unary RPC: метод, длительность и x-request-id.
func UnaryServerLogger(log *zap.Logger) grpc.UnaryServerInterceptor {
	if log == nil {
		log = zap.NewNop()
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)

		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("duration", time.Since(start)),
			zap.String("request_id", RequestIDFromContext(ctx)),
		}
		if err != nil {
			fields = append(fields, zap.String("grpc_code", status.Code(err).String()), zap.Error(err))
			log.Warn("grpc request failed", fields...)
			return resp, err
		}

		log.Info("grpc request completed", fields...)
		return resp, nil
	}
}
