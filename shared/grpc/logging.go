package grpc

import (
	"context"
	"time"

	"github.com/exchange-grpc/shared/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
		rpcLog := logger.WithTrace(ctx, log)

		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.String("request_id", RequestIDFromContext(ctx)),
			zap.Duration("duration", time.Since(start)),
		}
		if err != nil {
			logRPCError(rpcLog, "grpc request failed", err, fields)
			return resp, err
		}

		rpcLog.Debug("grpc request", fields...)
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
		rpcLog := logger.WithTrace(ctx, log)
		rpcLog.Debug("grpc stream started",
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
			logRPCError(rpcLog, "grpc stream failed", err, fields)
			return err
		}

		rpcLog.Debug("grpc stream completed", fields...)
		return nil
	}
}

func logRPCError(log *zap.Logger, msg string, err error, fields []zap.Field) {
	code := codes.Unknown
	if st, ok := status.FromError(err); ok {
		code = st.Code()
		fields = append(fields, zap.String("grpc_code", code.String()))
	}
	fields = append(fields, zap.Error(err))

	switch rpcErrorLogLevel(code) {
	case zapcore.DebugLevel:
		log.Debug(msg, fields...)
	case zapcore.ErrorLevel:
		log.Error(msg, fields...)
	default:
		log.Warn(msg, fields...)
	}
}

// rpcErrorLogLevel отделяет ожидаемые клиентские ошибки от сбоев инфраструктуры.
// Клиентские коды — Warn, чтобы их было видно, но не путать с падением сервиса.
func rpcErrorLogLevel(code codes.Code) zapcore.Level {
	switch code {
	case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists,
		codes.PermissionDenied, codes.Unauthenticated, codes.ResourceExhausted,
		codes.FailedPrecondition, codes.Canceled, codes.DeadlineExceeded:
		return zapcore.WarnLevel
	case codes.Internal, codes.Unavailable, codes.DataLoss, codes.Unknown:
		return zapcore.ErrorLevel
	default:
		return zapcore.WarnLevel
	}
}
