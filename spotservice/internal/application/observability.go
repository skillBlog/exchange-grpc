package application

import (
	"context"

	sharedgrpc "github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/shared/tracing"
	"go.uber.org/zap"
)

func logAudit(ctx context.Context, log *zap.Logger, msg, userID string, extra ...zap.Field) {
	sharedgrpc.LogAudit(ctx, log, msg, userID, extra...)
}

func logWarn(ctx context.Context, log *zap.Logger, msg, userID string, extra ...zap.Field) {
	sharedgrpc.LogWarn(ctx, log, msg, userID, extra...)
}

func checkViewMarketsRateLimit(ctx context.Context, limiter ViewMarketsRateLimiter, userID string) (err error) {
	if limiter == nil || userID == "" {
		return nil
	}
	ctx, span := tracing.Start(ctx, "spot.viewMarketsRateLimit")
	defer tracing.End(span, &err)
	return limiter.Allow(ctx, userID)
}
