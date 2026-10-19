package ratelimit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
)

func TestCreateOrderLimiter_deletesEmptyUserKeys(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	l := NewCreateOrderLimiter(application.CreateOrderRateLimitConfig{
		GlobalLimit:  100,
		GlobalWindow: time.Minute,
		BasicLimit:   10,
		UserWindow:   time.Minute,
	})
	l.now = func() time.Time { return now }

	if err := l.Allow(context.Background(), "user-1", nil); err != nil {
		t.Fatalf("Allow() error = %v", err)
	}
	if _, ok := l.users["user-1"]; !ok {
		t.Fatal("expected user key after allow")
	}

	users := map[string]counter{
		"empty": {timestamps: nil},
		"kept":  {timestamps: []time.Time{now}},
	}
	storeUserCounter(users, "empty", counter{})
	if _, ok := users["empty"]; ok {
		t.Fatal("empty user key must be deleted")
	}
	storeUserCounter(users, "kept", users["kept"])
	if _, ok := users["kept"]; !ok {
		t.Fatal("non-empty user key must be kept")
	}

	expired := now.Add(-2 * time.Minute)
	l.users["user-expired"] = counter{timestamps: []time.Time{expired}}
	later := now.Add(2 * time.Minute)
	l.now = func() time.Time { return later }
	if err := l.Allow(context.Background(), "user-expired", nil); err != nil {
		t.Fatalf("Allow() after window error = %v", err)
	}
	got, ok := l.users["user-expired"]
	if !ok {
		t.Fatal("expected user key after allow in new window")
	}
	if len(got.timestamps) == 0 {
		t.Fatal("user key must not hold empty timestamps")
	}
	if len(got.timestamps) != 1 || !got.timestamps[0].Equal(later) {
		t.Fatalf("timestamps = %v, want [%v]", got.timestamps, later)
	}
}

func TestCreateOrderLimiter_fullMapKeepsActiveCounter(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	limiter := NewCreateOrderLimiter(application.CreateOrderRateLimitConfig{
		GlobalLimit:  100,
		GlobalWindow: time.Minute,
		BasicLimit:   2,
		UserWindow:   time.Minute,
	})
	limiter.maxUsers = 1
	limiter.now = func() time.Time { return now }

	if err := limiter.Allow(context.Background(), "oldest", nil); err != nil {
		t.Fatalf("Allow(oldest) error = %v", err)
	}
	if err := limiter.Allow(context.Background(), "new", nil); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("Allow(new) error = %v, want ErrRateLimited", err)
	}
	now = now.Add(time.Second)
	if err := limiter.Allow(context.Background(), "oldest", nil); err != nil {
		t.Fatalf("second Allow(oldest) error = %v", err)
	}

	if _, ok := limiter.users["new"]; ok {
		t.Fatal("rejected user must not be stored")
	}
	if got := len(limiter.users["oldest"].timestamps); got != 2 {
		t.Fatalf("oldest attempts = %d, want 2", got)
	}
	if got := len(limiter.global.timestamps); got != 2 {
		t.Fatalf("global attempts = %d, want 2", got)
	}
}

func TestCreateOrderLimiter_expiredUserFreesSlot(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	limiter := NewCreateOrderLimiter(application.CreateOrderRateLimitConfig{
		GlobalLimit:  100,
		GlobalWindow: time.Minute,
		BasicLimit:   10,
		UserWindow:   time.Minute,
	})
	limiter.maxUsers = 1
	limiter.now = func() time.Time { return now }

	if err := limiter.Allow(context.Background(), "expired", nil); err != nil {
		t.Fatalf("Allow(expired) error = %v", err)
	}
	now = now.Add(2 * time.Minute)
	if err := limiter.Allow(context.Background(), "fresh", nil); err != nil {
		t.Fatalf("Allow(fresh) error = %v", err)
	}
	if _, ok := limiter.users["expired"]; ok {
		t.Fatal("expected expired user to be swept")
	}
	if _, ok := limiter.users["fresh"]; !ok {
		t.Fatal("expected fresh user to be stored")
	}
}

func TestCreateOrderLimiter_cancelledContextDoesNotCount(t *testing.T) {
	limiter := NewCreateOrderLimiter(application.CreateOrderRateLimitConfig{
		GlobalLimit:  1,
		GlobalWindow: time.Minute,
		BasicLimit:   1,
		UserWindow:   time.Minute,
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := limiter.Allow(ctx, "user-1", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("Allow() error = %v, want Canceled", err)
	}
	if err := limiter.Allow(context.Background(), "user-1", nil); err != nil {
		t.Fatalf("live Allow() error = %v", err)
	}
}

func TestRedisCreateOrderLimiter_cancelledContextSkipsRedis(t *testing.T) {
	limiter := NewRedisCreateOrderLimiter(nil, application.CreateOrderRateLimitConfig{
		GlobalLimit:  1,
		GlobalWindow: time.Minute,
		BasicLimit:   1,
		UserWindow:   time.Minute,
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := limiter.Allow(ctx, "user-1", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("Allow() error = %v", err)
	}
}
