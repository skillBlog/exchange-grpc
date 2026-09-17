package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/exchange-grpc/userservice/internal/application"
	"github.com/exchange-grpc/userservice/internal/domain"
)

const defaultMaxTrackedEmails = 10_000

// LoginLimiter ограничивает число попыток входа по email в заданном окне (in-memory).
type LoginLimiter struct {
	mu          sync.Mutex
	attempts    map[string][]time.Time
	maxAttempts int
	maxEmails   int
	window      time.Duration
	now         func() time.Time
	lastSweep   time.Time
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
		maxEmails:   defaultMaxTrackedEmails,
		window:      window,
		now:         time.Now,
	}
}

// Allow возвращает ошибку, если лимит попыток исчерпан.
func (l *LoginLimiter) Allow(ctx context.Context, email string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	cutoff := now.Add(-l.window)

	if l.lastSweep.IsZero() || now.Sub(l.lastSweep) >= l.window {
		l.sweepLocked(cutoff)
		l.lastSweep = now
	}

	history := l.attempts[email]
	active := history[:0]
	for _, ts := range history {
		if ts.After(cutoff) {
			active = append(active, ts)
		}
	}

	if len(active) == 0 {
		delete(l.attempts, email)
	}

	if len(active) >= l.maxAttempts {
		l.attempts[email] = active
		return fmt.Errorf("%w: too many login attempts", domain.ErrRateLimited)
	}

	l.evictIfOverCapLocked(email, cutoff)
	l.attempts[email] = append(active, now)
	return nil
}

func (l *LoginLimiter) sweepLocked(cutoff time.Time) {
	for email, history := range l.attempts {
		kept := history[:0]
		for _, ts := range history {
			if ts.After(cutoff) {
				kept = append(kept, ts)
			}
		}
		if len(kept) == 0 {
			delete(l.attempts, email)
			continue
		}
		l.attempts[email] = kept
	}
}

func (l *LoginLimiter) evictIfOverCapLocked(current string, cutoff time.Time) {
	maxEmails := l.maxEmails
	if maxEmails <= 0 {
		maxEmails = defaultMaxTrackedEmails
	}
	if len(l.attempts) < maxEmails {
		return
	}
	l.sweepLocked(cutoff)
	if len(l.attempts) < maxEmails {
		return
	}
	for email := range l.attempts {
		if email != current {
			delete(l.attempts, email)
			return
		}
	}
}

var _ application.LoginRateLimiter = (*LoginLimiter)(nil)
