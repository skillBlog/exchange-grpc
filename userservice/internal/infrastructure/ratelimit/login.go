package ratelimit

import (
	"sync"
	"time"

	"github.com/exchange-grpc/userservice/internal/application"
)

// LoginLimiter ограничивает число попыток входа по email в заданном окне.
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

// Allow возвращает true, если попытка входа разрешена.
func (l *LoginLimiter) Allow(email string) bool {
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
		return false
	}

	l.attempts[email] = append(active, now)
	return true
}

var _ application.LoginRateLimiter = (*LoginLimiter)(nil)
