package ratelimit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/exchange-grpc/spotservice/internal/domain"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestViewMarketsLimiter_cancelledContextDoesNotCount(t *testing.T) {
	limiter := NewViewMarketsLimiter(1, time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := limiter.Allow(ctx, "user-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Allow() error = %v, want Canceled", err)
	}
	if err := limiter.Allow(context.Background(), "user-1"); err != nil {
		t.Fatalf("live Allow() error = %v", err)
	}
}

func TestRedisViewMarketsLimiter_cancelledContextSkipsRedis(t *testing.T) {
	limiter := NewRedisViewMarketsLimiter(nil, 1, time.Minute, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := limiter.Allow(ctx, "user-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Allow() error = %v, want Canceled", err)
	}
}

func TestRedisViewMarketsLimiter_fallsBackToMemoryOnRedisError(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		MaxRetries:  -1,
		DialTimeout: 50 * time.Millisecond,
		ReadTimeout: 50 * time.Millisecond,
	})
	t.Cleanup(func() { _ = client.Close() })

	core, logs := observer.New(zapcore.ErrorLevel)
	limiter := NewRedisViewMarketsLimiter(client, 2, time.Minute, zap.New(core))

	if err := limiter.Allow(context.Background(), "user-1"); err != nil {
		t.Fatalf("first Allow() error = %v", err)
	}
	if err := limiter.Allow(context.Background(), "user-1"); err != nil {
		t.Fatalf("second Allow() error = %v", err)
	}
	if err := limiter.Allow(context.Background(), "user-1"); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("third Allow() error = %v, want ErrRateLimited", err)
	}

	entries := logs.FilterMessage("redis view markets limiter failed, using in-memory fallback").All()
	if len(entries) != 1 {
		t.Fatalf("fallback logs = %d, want 1", len(entries))
	}
}

func TestRedisViewMarketsLimiter_nilClientUsesMemory(t *testing.T) {
	limiter := NewRedisViewMarketsLimiter(nil, 1, time.Minute, nil)

	if err := limiter.Allow(context.Background(), "user-1"); err != nil {
		t.Fatalf("first Allow() error = %v", err)
	}
	if err := limiter.Allow(context.Background(), "user-1"); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("second Allow() error = %v, want ErrRateLimited", err)
	}
}
