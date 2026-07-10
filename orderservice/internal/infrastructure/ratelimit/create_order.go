package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
)

type counter struct {
	timestamps []time.Time
}

// CreateOrderLimiter ограничивает CreateOrder in-memory (глобально и per-user).
type CreateOrderLimiter struct {
	mu     sync.Mutex
	cfg    application.CreateOrderRateLimitConfig
	global counter
	users  map[string]counter
	now    func() time.Time
}

// NewCreateOrderLimiter создаёт in-memory rate limiter для CreateOrder.
func NewCreateOrderLimiter(cfg application.CreateOrderRateLimitConfig) *CreateOrderLimiter {
	if cfg.GlobalLimit <= 0 {
		cfg.GlobalLimit = 20000
	}
	if cfg.GlobalWindow <= 0 {
		cfg.GlobalWindow = time.Minute
	}
	if cfg.BasicLimit <= 0 {
		cfg.BasicLimit = 10
	}
	if cfg.PremiumLimit <= 0 {
		cfg.PremiumLimit = 100
	}
	if cfg.AdminLimit <= 0 {
		cfg.AdminLimit = 1000
	}
	if cfg.UserWindow <= 0 {
		cfg.UserWindow = time.Minute
	}
	return &CreateOrderLimiter{
		cfg:   cfg,
		users: make(map[string]counter),
		now:   time.Now,
	}
}

// Allow проверяет глобальный и per-user лимиты.
func (l *CreateOrderLimiter) Allow(_ context.Context, userID string, userRoles []string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if !allowCounter(&l.global, now, l.cfg.GlobalLimit, l.cfg.GlobalWindow) {
		return fmt.Errorf("%w: global create order limit exceeded", domain.ErrRateLimited)
	}

	tier := application.CreateOrderRateTierFromRoles(userRoles)
	userCounter := l.users[userID]
	if !allowCounter(&userCounter, now, l.cfg.LimitForTier(tier), l.cfg.UserWindow) {
		l.users[userID] = userCounter
		return fmt.Errorf("%w: too many create order requests", domain.ErrRateLimited)
	}
	l.users[userID] = userCounter
	return nil
}

func allowCounter(counter *counter, now time.Time, limit int, window time.Duration) bool {
	cutoff := now.Add(-window)
	active := counter.timestamps[:0]
	for _, ts := range counter.timestamps {
		if ts.After(cutoff) {
			active = append(active, ts)
		}
	}
	if len(active) >= limit {
		counter.timestamps = active
		return false
	}
	counter.timestamps = append(active, now)
	return true
}

var _ application.CreateOrderRateLimiter = (*CreateOrderLimiter)(nil)
