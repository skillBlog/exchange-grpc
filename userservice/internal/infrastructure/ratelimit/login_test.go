package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/exchange-grpc/userservice/internal/domain"
)

func TestLoginLimiter_uniqueEmailsAreCapped(t *testing.T) {
	limiter := &LoginLimiter{
		attempts:    make(map[string][]time.Time),
		maxAttempts: 5,
		maxEmails:   8,
		window:      time.Minute,
		now:         time.Now,
	}

	for i := 0; i < 8; i++ {
		email := "user" + strconv.Itoa(i) + "@example.com"
		if err := limiter.Allow(context.Background(), email); err != nil {
			t.Fatalf("Allow(%s) error = %v", email, err)
		}
	}
	err := limiter.Allow(context.Background(), "overflow@example.com")
	if !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("overflow Allow() error = %v, want ErrRateLimited", err)
	}

	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if len(limiter.attempts) != 8 {
		t.Fatalf("tracked emails = %d, want 8", len(limiter.attempts))
	}
	if _, ok := limiter.attempts["overflow@example.com"]; ok {
		t.Fatal("rejected email must not be stored")
	}
	if _, ok := limiter.attempts["user0@example.com"]; !ok {
		t.Fatal("oldest active email must stay tracked")
	}
}

func TestLoginLimiter_expiredEntriesAreRemoved(t *testing.T) {
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	limiter := &LoginLimiter{
		attempts:    make(map[string][]time.Time),
		maxAttempts: 5,
		maxEmails:   100,
		window:      time.Minute,
		now:         func() time.Time { return clock },
	}

	if err := limiter.Allow(context.Background(), "one@example.com"); err != nil {
		t.Fatalf("Allow() error = %v", err)
	}
	clock = clock.Add(2 * time.Minute)
	if err := limiter.Allow(context.Background(), "two@example.com"); err != nil {
		t.Fatalf("second Allow() error = %v", err)
	}

	limiter.mu.Lock()
	_, stale := limiter.attempts["one@example.com"]
	n := len(limiter.attempts)
	limiter.mu.Unlock()
	if stale {
		t.Fatal("expected expired email to be swept")
	}
	if n != 1 {
		t.Fatalf("tracked emails = %d, want 1", n)
	}
}

func TestLoginLimiter_fullMapKeepsActiveCounter(t *testing.T) {
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	limiter := &LoginLimiter{
		attempts:    make(map[string][]time.Time),
		maxAttempts: 2,
		maxEmails:   1,
		window:      time.Minute,
		now:         func() time.Time { return clock },
	}

	if err := limiter.Allow(context.Background(), "oldest@example.com"); err != nil {
		t.Fatalf("Allow(oldest) error = %v", err)
	}
	if err := limiter.Allow(context.Background(), "new@example.com"); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("Allow(new) error = %v, want ErrRateLimited", err)
	}
	clock = clock.Add(time.Second)
	if err := limiter.Allow(context.Background(), "oldest@example.com"); err != nil {
		t.Fatalf("second Allow(oldest) error = %v", err)
	}
	err := limiter.Allow(context.Background(), "oldest@example.com")
	if !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("third Allow(oldest) error = %v, want ErrRateLimited", err)
	}

	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if len(limiter.attempts) != 1 {
		t.Fatalf("tracked emails = %d, want 1", len(limiter.attempts))
	}
	if got := len(limiter.attempts["oldest@example.com"]); got != 2 {
		t.Fatalf("oldest attempts = %d, want 2", got)
	}
}

func TestLoginLimiter_stillRateLimitsSameEmail(t *testing.T) {
	limiter := NewLoginLimiter(1, time.Minute)
	if err := limiter.Allow(context.Background(), "same@example.com"); err != nil {
		t.Fatalf("first Allow() error = %v", err)
	}
	err := limiter.Allow(context.Background(), "same@example.com")
	if !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("second Allow() error = %v, want ErrRateLimited", err)
	}
}

func TestLoginLimiter_cancelledContextDoesNotCount(t *testing.T) {
	limiter := NewLoginLimiter(1, time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := limiter.Allow(ctx, "same@example.com"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Allow() error = %v, want Canceled", err)
	}
	if err := limiter.Allow(context.Background(), "same@example.com"); err != nil {
		t.Fatalf("live Allow() error = %v", err)
	}
}

func TestRedisLoginLimiter_cancelledContextSkipsRedis(t *testing.T) {
	limiter := NewRedisLoginLimiter(nil, 1, time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := limiter.Allow(ctx, "same@example.com"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Allow() error = %v, want Canceled", err)
	}
}

func TestLoginLimiter_concurrentUniqueEmailsStayBounded(t *testing.T) {
	limiter := &LoginLimiter{
		attempts:    make(map[string][]time.Time),
		maxAttempts: 50,
		maxEmails:   16,
		window:      time.Minute,
		now:         time.Now,
	}

	var wg sync.WaitGroup
	for i := 0; i < 80; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = limiter.Allow(context.Background(), fmt.Sprintf("u%d@example.com", i))
		}(i)
	}
	wg.Wait()

	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if len(limiter.attempts) > 16 {
		t.Fatalf("tracked emails = %d, want <= 16", len(limiter.attempts))
	}
	for email, history := range limiter.attempts {
		if len(history) == 0 {
			t.Fatalf("email %s has an empty history", email)
		}
	}
}
