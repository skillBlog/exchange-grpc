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

func TestViewMarketsLimiter_fullMapKeepsActiveCounter(t *testing.T) {
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	limiter := NewViewMarketsLimiter(2, time.Minute)
	limiter.maxUsers = 1
	limiter.now = func() time.Time { return clock }

	if err := limiter.Allow(context.Background(), "oldest"); err != nil {
		t.Fatalf("Allow(oldest) error = %v", err)
	}
	if err := limiter.Allow(context.Background(), "new"); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("Allow(new) error = %v, want ErrRateLimited", err)
	}
	clock = clock.Add(time.Second)
	if err := limiter.Allow(context.Background(), "oldest"); err != nil {
		t.Fatalf("second Allow(oldest) error = %v", err)
	}

	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if _, ok := limiter.attempts["new"]; ok {
		t.Fatal("rejected user must not be stored")
	}
	if got := len(limiter.attempts["oldest"]); got != 2 {
		t.Fatalf("oldest attempts = %d, want 2", got)
	}
}

func TestViewMarketsLimiter_expiredUserFreesSlot(t *testing.T) {
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	limiter := NewViewMarketsLimiter(5, time.Minute)
	limiter.maxUsers = 1
	limiter.now = func() time.Time { return clock }

	if err := limiter.Allow(context.Background(), "expired"); err != nil {
		t.Fatalf("Allow(expired) error = %v", err)
	}
	clock = clock.Add(2 * time.Minute)
	if err := limiter.Allow(context.Background(), "fresh"); err != nil {
		t.Fatalf("Allow(fresh) error = %v", err)
	}

	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if _, ok := limiter.attempts["expired"]; ok {
		t.Fatal("expected expired user to be swept")
	}
	if _, ok := limiter.attempts["fresh"]; !ok {
		t.Fatal("expected fresh user to be stored")
	}
}

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
	limiter := NewRedisViewMarketsLimiter(nil, 1, time.Minute, nil, true)
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
	limiter := NewRedisViewMarketsLimiter(client, 2, time.Minute, zap.New(core), true)

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

func TestRedisViewMarketsLimiter_productionDoesNotUseMemory(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		MaxRetries:  -1,
		DialTimeout: 50 * time.Millisecond,
		ReadTimeout: 50 * time.Millisecond,
	})
	t.Cleanup(func() { _ = client.Close() })

	core, logs := observer.New(zapcore.ErrorLevel)
	limiter := NewRedisViewMarketsLimiter(client, 5, time.Minute, zap.New(core), false)

	err := limiter.Allow(context.Background(), "user-1")
	if err == nil || errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("Allow() error = %v, want redis failure", err)
	}
	if err := limiter.Allow(context.Background(), "user-2"); err == nil || errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("second Allow() error = %v, want redis failure", err)
	}

	if logs.FilterMessage("redis view markets limiter failed, using in-memory fallback").Len() != 0 {
		t.Fatal("production must not log an in-memory fallback")
	}
	if logs.FilterMessage("redis view markets limiter failed").Len() != 1 {
		t.Fatalf("redis failure logs = %d, want 1", logs.FilterMessage("redis view markets limiter failed").Len())
	}
}

func TestRedisViewMarketsLimiter_nilClientUsesMemory(t *testing.T) {
	limiter := NewRedisViewMarketsLimiter(nil, 1, time.Minute, nil, true)

	if err := limiter.Allow(context.Background(), "user-1"); err != nil {
		t.Fatalf("first Allow() error = %v", err)
	}
	if err := limiter.Allow(context.Background(), "user-1"); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("second Allow() error = %v, want ErrRateLimited", err)
	}
}
