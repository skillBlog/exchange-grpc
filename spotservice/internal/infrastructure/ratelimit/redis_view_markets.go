package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/exchange-grpc/spotservice/internal/application"
	"github.com/exchange-grpc/spotservice/internal/domain"
	"github.com/redis/go-redis/v9"
)

var viewMarketsAllowScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
if current > tonumber(ARGV[2]) then
  return 0
end
return 1
`)

// RedisViewMarketsLimiter ограничивает ViewMarkets через Redis.
type RedisViewMarketsLimiter struct {
	client      *redis.Client
	maxAttempts int
	window      time.Duration
	prefix      string
}

// NewRedisViewMarketsLimiter создаёт распределённый rate limiter для ViewMarkets.
func NewRedisViewMarketsLimiter(client *redis.Client, maxAttempts int, window time.Duration) *RedisViewMarketsLimiter {
	if maxAttempts <= 0 {
		maxAttempts = 30
	}
	if window <= 0 {
		window = time.Minute
	}
	return &RedisViewMarketsLimiter{
		client:      client,
		maxAttempts: maxAttempts,
		window:      window,
		prefix:      "spotservice:ratelimit:view_markets",
	}
}

// Allow проверяет лимит вызовов ViewMarkets по user_id.
func (l *RedisViewMarketsLimiter) Allow(ctx context.Context, userID string) error {
	key := fmt.Sprintf("%s:%s", l.prefix, userID)
	result, err := viewMarketsAllowScript.Run(
		ctx,
		l.client,
		[]string{key},
		l.window.Milliseconds(),
		l.maxAttempts,
	).Int()
	if err != nil {
		return fmt.Errorf("check view markets rate limit: %w", err)
	}
	if result != 1 {
		return fmt.Errorf("%w: too many requests", domain.ErrRateLimited)
	}
	return nil
}

var _ application.ViewMarketsRateLimiter = (*RedisViewMarketsLimiter)(nil)
