package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/exchange-grpc/userservice/internal/application"
	"github.com/exchange-grpc/userservice/internal/domain"
	"github.com/redis/go-redis/v9"
)

var loginAllowScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
if current > tonumber(ARGV[2]) then
  return 0
end
return 1
`)

// RedisLoginLimiter ограничивает Login через Redis (общий лимит между подами).
type RedisLoginLimiter struct {
	client      *redis.Client
	maxAttempts int
	window      time.Duration
	prefix      string
}

// NewRedisLoginLimiter создаёт распределённый login rate limiter.
func NewRedisLoginLimiter(client *redis.Client, maxAttempts int, window time.Duration) *RedisLoginLimiter {
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	if window <= 0 {
		window = time.Minute
	}
	return &RedisLoginLimiter{
		client:      client,
		maxAttempts: maxAttempts,
		window:      window,
		prefix:      "userservice:ratelimit:login",
	}
}

// Allow проверяет лимит попыток входа по email.
func (l *RedisLoginLimiter) Allow(ctx context.Context, email string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%s", l.prefix, email)
	result, err := loginAllowScript.Run(
		ctx,
		l.client,
		[]string{key},
		l.window.Milliseconds(),
		l.maxAttempts,
	).Int()
	if err != nil {
		return fmt.Errorf("check login rate limit: %w", err)
	}
	if result != 1 {
		return fmt.Errorf("%w: too many login attempts", domain.ErrRateLimited)
	}
	return nil
}

var _ application.LoginRateLimiter = (*RedisLoginLimiter)(nil)
