package ratelimit

import (
	"sync"
	"time"

	"github.com/exchange-grpc/spotservice/internal/application"
)

// ViewMarketsLimiter ограничивает частоту ViewMarkets по user_id.
type ViewMarketsLimiter struct {
	mu          sync.Mutex
	attempts    map[string][]time.Time
	maxAttempts int
	window      time.Duration
	now         func() time.Time
}

// NewViewMarketsLimiter создаёт rate limiter для ViewMarkets.
func NewViewMarketsLimiter(maxAttempts int, window time.Duration) *ViewMarketsLimiter {
	if maxAttempts <= 0 {
		maxAttempts = 30
	}
	if window <= 0 {
		window = time.Minute
	}
	return &ViewMarketsLimiter{
		attempts:    make(map[string][]time.Time),
		maxAttempts: maxAttempts,
		window:      window,
		now:         time.Now,
	}
}

// Allow возвращает true, если запрос разрешён.
func (l *ViewMarketsLimiter) Allow(userID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	cutoff := now.Add(-l.window)

	history := l.attempts[userID]
	active := history[:0]
	for _, ts := range history {
		if ts.After(cutoff) {
			active = append(active, ts)
		}
	}

	if len(active) >= l.maxAttempts {
		l.attempts[userID] = active
		return false
	}

	l.attempts[userID] = append(active, now)
	return true
}

var _ application.ViewMarketsRateLimiter = (*ViewMarketsLimiter)(nil)
