package ratelimit

import (
	"context"
	"errors"
	"testing"
	"time"
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
	limiter := NewRedisViewMarketsLimiter(nil, 1, time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := limiter.Allow(ctx, "user-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Allow() error = %v, want Canceled", err)
	}
}
