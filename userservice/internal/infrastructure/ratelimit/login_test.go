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

	for i := 0; i < 40; i++ {
		email := "user" + strconv.Itoa(i) + "@example.com"
		if err := limiter.Allow(context.Background(), email); err != nil {
			t.Fatalf("Allow(%s) error = %v", email, err)
		}
	}

	limiter.mu.Lock()
	n := len(limiter.attempts)
	limiter.mu.Unlock()
	if n > 8 {
		t.Fatalf("tracked emails = %d, want <= 8", n)
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
	n := len(limiter.attempts)
	limiter.mu.Unlock()
	if n > 16 {
		t.Fatalf("tracked emails = %d, want <= 16", n)
	}
}
