package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/exchange-grpc/spotservice/internal/application"
	"github.com/exchange-grpc/spotservice/internal/domain"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const fallbackLogInterval = 10 * time.Second

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
// При ошибке соединения запросы уходят в in-memory limiter, чтобы сервис
// не отвечал одной и той же ошибкой Redis на каждый ViewMarkets.
type RedisViewMarketsLimiter struct {
	client      *redis.Client
	fallback    *ViewMarketsLimiter
	log         *zap.Logger
	maxAttempts int
	window      time.Duration
	prefix      string

	fallbackLogMu   sync.Mutex
	lastFallbackLog time.Time
}

// NewRedisViewMarketsLimiter создаёт распределённый rate limiter для ViewMarkets.
func NewRedisViewMarketsLimiter(client *redis.Client, maxAttempts int, window time.Duration, log *zap.Logger) *RedisViewMarketsLimiter {
	if maxAttempts <= 0 {
		maxAttempts = 30
	}
	if window <= 0 {
		window = time.Minute
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &RedisViewMarketsLimiter{
		client:      client,
		fallback:    NewViewMarketsLimiter(maxAttempts, window),
		log:         log,
		maxAttempts: maxAttempts,
		window:      window,
		prefix:      "spotservice:ratelimit:view_markets",
	}
}

// Allow проверяет лимит вызовов ViewMarkets по user_id.
func (l *RedisViewMarketsLimiter) Allow(ctx context.Context, userID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if l.client == nil {
		return l.fallback.Allow(ctx, userID)
	}

	key := fmt.Sprintf("%s:%s", l.prefix, userID)
	result, err := viewMarketsAllowScript.Run(
		ctx,
		l.client,
		[]string{key},
		l.window.Milliseconds(),
		l.maxAttempts,
	).Int()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		l.logFallback(err)
		return l.fallback.Allow(ctx, userID)
	}
	if result != 1 {
		return fmt.Errorf("%w: too many requests", domain.ErrRateLimited)
	}
	return nil
}

func (l *RedisViewMarketsLimiter) logFallback(err error) {
	now := time.Now()
	l.fallbackLogMu.Lock()
	defer l.fallbackLogMu.Unlock()
	if !l.lastFallbackLog.IsZero() && now.Sub(l.lastFallbackLog) < fallbackLogInterval {
		return
	}
	l.lastFallbackLog = now
	l.log.Error("redis view markets limiter failed, using in-memory fallback", zap.Error(err))
}

var _ application.ViewMarketsRateLimiter = (*RedisViewMarketsLimiter)(nil)
