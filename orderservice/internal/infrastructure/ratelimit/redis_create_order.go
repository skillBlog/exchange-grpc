package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/redis/go-redis/v9"
)

var allowScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
if current > tonumber(ARGV[2]) then
  return 0
end
return 1
`)

// RedisCreateOrderLimiter ограничивает CreateOrder через Redis.
type RedisCreateOrderLimiter struct {
	client *redis.Client
	cfg    application.CreateOrderRateLimitConfig
	prefix string
}

// NewRedisCreateOrderLimiter создаёт Redis rate limiter для CreateOrder.
func NewRedisCreateOrderLimiter(client *redis.Client, cfg application.CreateOrderRateLimitConfig) *RedisCreateOrderLimiter {
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
	return &RedisCreateOrderLimiter{
		client: client,
		cfg:    cfg,
		prefix: "orderservice:ratelimit:create_order",
	}
}

// Allow проверяет глобальный и per-user лимиты.
func (l *RedisCreateOrderLimiter) Allow(ctx context.Context, userID string, userRoles []string) error {
	ok, err := l.allowKey(ctx, l.prefix+":global", l.cfg.GlobalLimit, l.cfg.GlobalWindow)
	if err != nil {
		return fmt.Errorf("check global rate limit: %w", err)
	}
	if !ok {
		return fmt.Errorf("%w: global create order limit exceeded", domain.ErrRateLimited)
	}

	tier := application.CreateOrderRateTierFromRoles(userRoles)
	userKey := fmt.Sprintf("%s:user:%s", l.prefix, userID)
	ok, err = l.allowKey(ctx, userKey, l.cfg.LimitForTier(tier), l.cfg.UserWindow)
	if err != nil {
		return fmt.Errorf("check user rate limit: %w", err)
	}
	if !ok {
		return fmt.Errorf("%w: too many create order requests", domain.ErrRateLimited)
	}
	return nil
}

func (l *RedisCreateOrderLimiter) allowKey(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	result, err := allowScript.Run(
		ctx,
		l.client,
		[]string{key},
		window.Milliseconds(),
		limit,
	).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

var _ application.CreateOrderRateLimiter = (*RedisCreateOrderLimiter)(nil)
