package grpc

import (
	"context"

	"github.com/exchange-grpc/shared/logger"
	"go.uber.org/zap"
)

// LogAudit пишет Info-аудит с user_id и request_id из контекста.
func LogAudit(ctx context.Context, log *zap.Logger, msg, userID string, extra ...zap.Field) {
	logAt(ctx, log, (*zap.Logger).Info, msg, userID, extra...)
}

// LogWarn пишет Warn с user_id и request_id из контекста.
func LogWarn(ctx context.Context, log *zap.Logger, msg, userID string, extra ...zap.Field) {
	logAt(ctx, log, (*zap.Logger).Warn, msg, userID, extra...)
}

// LogError пишет Error с user_id и request_id из контекста.
func LogError(ctx context.Context, log *zap.Logger, msg, userID string, extra ...zap.Field) {
	logAt(ctx, log, (*zap.Logger).Error, msg, userID, extra...)
}

func logAt(
	ctx context.Context,
	log *zap.Logger,
	write func(*zap.Logger, string, ...zap.Field),
	msg, userID string,
	extra ...zap.Field,
) {
	if log == nil {
		return
	}
	write(logger.WithTrace(ctx, log), msg, auditFields(ctx, userID, extra...)...)
}

func auditFields(ctx context.Context, userID string, extra ...zap.Field) []zap.Field {
	fields := make([]zap.Field, 0, 2+len(extra))
	if userID != "" {
		fields = append(fields, zap.String("user_id", userID))
	}
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	return append(fields, extra...)
}
