package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
)

const defaultMaxTrackedUsers = 10_000

type counter struct {
	timestamps []time.Time
}

// CreateOrderLimiter ограничивает CreateOrder in-memory (глобально и per-user).
type CreateOrderLimiter struct {
	mu       sync.Mutex
	cfg      application.CreateOrderRateLimitConfig
	global   counter
	users    map[string]counter
	maxUsers int
	now      func() time.Time
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
		cfg:      cfg,
		users:    make(map[string]counter),
		maxUsers: defaultMaxTrackedUsers,
		now:      time.Now,
	}
}

// Allow проверяет глобальный и per-user лимиты.
func (l *CreateOrderLimiter) Allow(ctx context.Context, userID string, userRoles []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweepUsersLocked(now)
	if _, exists := l.users[userID]; !exists && len(l.users) >= l.userCap() {
		return fmt.Errorf("%w: too many create order requests", domain.ErrRateLimited)
	}
	if !allowCounter(&l.global, now, l.cfg.GlobalLimit, l.cfg.GlobalWindow) {
		return fmt.Errorf("%w: global create order limit exceeded", domain.ErrRateLimited)
	}

	tier := application.CreateOrderRateTierFromRoles(userRoles)
	userCounter := l.users[userID]
	if !allowCounter(&userCounter, now, l.cfg.LimitForTier(tier), l.cfg.UserWindow) {
		storeUserCounter(l.users, userID, userCounter)
		return fmt.Errorf("%w: too many create order requests", domain.ErrRateLimited)
	}
	storeUserCounter(l.users, userID, userCounter)
	return nil
}

func (l *CreateOrderLimiter) userCap() int {
	if l.maxUsers <= 0 {
		return defaultMaxTrackedUsers
	}
	return l.maxUsers
}

func (l *CreateOrderLimiter) sweepUsersLocked(now time.Time) {
	cutoff := now.Add(-l.cfg.UserWindow)
	for userID, userCounter := range l.users {
		kept := userCounter.timestamps[:0]
		for _, ts := range userCounter.timestamps {
			if ts.After(cutoff) {
				kept = append(kept, ts)
			}
		}
		if len(kept) == 0 {
			delete(l.users, userID)
			continue
		}
		userCounter.timestamps = kept
		l.users[userID] = userCounter
	}
}

func storeUserCounter(users map[string]counter, userID string, userCounter counter) {
	if len(userCounter.timestamps) == 0 {
		delete(users, userID)
		return
	}
	users[userID] = userCounter
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
