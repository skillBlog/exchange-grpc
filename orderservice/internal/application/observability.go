package application

import (
	"context"

	sharedgrpc "github.com/exchange-grpc/shared/grpc"
	"go.uber.org/zap"
)

func logAudit(ctx context.Context, log *zap.Logger, msg, userID string, extra ...zap.Field) {
	sharedgrpc.LogAudit(ctx, log, msg, userID, extra...)
}

func logError(ctx context.Context, log *zap.Logger, msg, userID string, extra ...zap.Field) {
	sharedgrpc.LogError(ctx, log, msg, userID, extra...)
}
