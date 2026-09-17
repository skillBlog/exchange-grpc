package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	sharedgrpc "github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/shared/tracing"
	"go.uber.org/zap"
)

func hashEmail(email string) string {
	sum := sha256.Sum256([]byte(email))
	return hex.EncodeToString(sum[:])
}

func logAudit(ctx context.Context, log *zap.Logger, msg, userID string, extra ...zap.Field) {
	sharedgrpc.LogAudit(ctx, log, msg, userID, extra...)
}

func hashPassword(ctx context.Context, hasher PasswordHasher, password string) (hash string, err error) {
	_, span := tracing.Start(ctx, "user.hashPassword")
	defer tracing.End(span, &err)
	return hasher.Hash(password)
}

func comparePassword(ctx context.Context, hasher PasswordHasher, hash, password string) (err error) {
	_, span := tracing.Start(ctx, "user.comparePassword")
	defer tracing.End(span, &err)
	return hasher.Compare(hash, password)
}

func checkLoginRateLimit(ctx context.Context, limiter LoginRateLimiter, email string) (err error) {
	if limiter == nil {
		return nil
	}
	_, span := tracing.Start(ctx, "user.loginRateLimit")
	defer tracing.End(span, &err)
	return limiter.Allow(ctx, email)
}
