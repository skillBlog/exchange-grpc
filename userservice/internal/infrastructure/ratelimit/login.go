package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/exchange-grpc/userservice/internal/application"
	"github.com/exchange-grpc/userservice/internal/domain"
)

// LoginLimiter ограничивает число попыток входа по email в заданном окне (in-memory).
type LoginLimiter struct {
	mu          sync.Mutex
	attempts    map[string][]time.Time
	maxAttempts int
	window      time.Duration
	now         func() time.Time
}

// NewLoginLimiter создаёт rate limiter для Login.
func NewLoginLimiter(maxAttempts int, window time.Duration) *LoginLimiter {
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	if window <= 0 {
		window = time.Minute
	}
	return &LoginLimiter{
		attempts:    make(map[string][]time.Time),
		maxAttempts: maxAttempts,
		window:      window,
		now:         time.Now,
	}
}

// Allow возвращает ошибку, если лимит попыток исчерпан.
func (l *LoginLimiter) Allow(_ context.Context, email string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	cutoff := now.Add(-l.window)

	history := l.attempts[email]
	active := history[:0]
	for _, ts := range history {
		if ts.After(cutoff) {
			active = append(active, ts)
		}
	}

	if len(active) >= l.maxAttempts {
		l.attempts[email] = active
		return fmt.Errorf("%w: too many login attempts", domain.ErrRateLimited)
	}

	l.attempts[email] = append(active, now)
	return nil
}

var _ application.LoginRateLimiter = (*LoginLimiter)(nil)
