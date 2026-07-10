package grpc

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryServerLogging логирует unary RPC-запросы.
func UnaryServerLogging(log *zap.Logger) grpc.UnaryServerInterceptor {
	if log == nil {
		log = zap.NewNop()
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)

		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.String("request_id", RequestIDFromContext(ctx)),
			zap.Duration("duration", time.Since(start)),
		}
		if err != nil {
			if st, ok := status.FromError(err); ok {
				fields = append(fields, zap.String("grpc_code", st.Code().String()))
			}
			fields = append(fields, zap.Error(err))
			log.Warn("grpc request failed", fields...)
			return resp, err
		}

		log.Info("grpc request", fields...)
		return resp, nil
	}
}

// StreamServerLogging логирует начало и завершение streaming RPC.
func StreamServerLogging(log *zap.Logger) grpc.StreamServerInterceptor {
	if log == nil {
		log = zap.NewNop()
	}

	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		ctx := stream.Context()
		log.Info("grpc stream started",
			zap.String("method", info.FullMethod),
			zap.String("request_id", RequestIDFromContext(ctx)),
		)

		err := handler(srv, stream)
		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.String("request_id", RequestIDFromContext(ctx)),
			zap.Duration("duration", time.Since(start)),
		}
		if err != nil {
			if st, ok := status.FromError(err); ok {
				fields = append(fields, zap.String("grpc_code", st.Code().String()))
			}
			fields = append(fields, zap.Error(err))
			log.Warn("grpc stream failed", fields...)
			return err
		}

		log.Info("grpc stream completed", fields...)
		return nil
	}
}
