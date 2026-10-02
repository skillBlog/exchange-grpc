package config

import (
	"os"
	"time"

	"github.com/exchange-grpc/shared/envparse"
)

const (
	defaultGRPCAddr              = ":50051"
	defaultJWTSecret             = "dev-exchange-secret"
	defaultAccessTokenTTL        = 15 * time.Minute
	defaultDatabaseURL           = "postgres://exchange:exchange@localhost:5432/spotservice?sslmode=disable"
	defaultMigrationsDir         = "migrations"
	defaultMarketCacheTTL        = 30 * time.Second
	defaultViewMarketsRateLimit  = 30
	defaultViewMarketsRateWindow = time.Minute
	defaultHealthCheckTimeout    = 3 * time.Second
	defaultRedisURL              = "redis://localhost:6379/0"
	defaultRedisPoolSize         = 10
	defaultRedisMaxRetries       = 3
	defaultMetricsAddr           = ":2112"
)

// Config содержит runtime-конфигурацию spotservice.
type Config struct {
	GRPCAddr              string
	JWTSecret             string
	AccessTokenTTL        time.Duration
	DatabaseURL           string
	MigrationsDir         string
	MarketCacheTTL        time.Duration
	ViewMarketsRateLimit  int
	ViewMarketsRateWindow time.Duration
	HealthCheckTimeout    time.Duration
	LogLevelAddr          string
	RedisURL              string
	RedisPoolSize         int
	RedisMaxRetries       int
	MetricsAddr           string
}

// LoadConfig читает конфигурацию из переменных окружения.
func LoadConfig() (Config, error) {
	var env envparse.Loader
	cfg := Config{
		GRPCAddr:              env.String("SPOT_SERVICE_ADDR", defaultGRPCAddr),
		JWTSecret:             env.String("JWT_SECRET", defaultJWTSecret),
		AccessTokenTTL:        env.Duration("JWT_ACCESS_TTL", defaultAccessTokenTTL),
		DatabaseURL:           env.String("SPOT_DATABASE_URL", defaultDatabaseURL),
		MigrationsDir:         env.String("SPOT_MIGRATIONS_DIR", defaultMigrationsDir),
		MarketCacheTTL:        env.Duration("MARKET_CACHE_TTL", defaultMarketCacheTTL),
		ViewMarketsRateLimit:  env.Int("VIEW_MARKETS_RATE_LIMIT", defaultViewMarketsRateLimit),
		ViewMarketsRateWindow: env.Duration("VIEW_MARKETS_RATE_WINDOW", defaultViewMarketsRateWindow),
		HealthCheckTimeout:    env.Duration("HEALTH_CHECK_TIMEOUT", defaultHealthCheckTimeout),
		LogLevelAddr:          os.Getenv("LOG_LEVEL_ADDR"),
		RedisURL:              env.String("REDIS_URL", defaultRedisURL),
		RedisPoolSize:         env.Int("REDIS_POOL_SIZE", defaultRedisPoolSize),
		RedisMaxRetries:       env.Int("REDIS_MAX_RETRIES", defaultRedisMaxRetries),
		MetricsAddr:           env.String("METRICS_ADDR", defaultMetricsAddr),
	}
	return cfg, env.Err()
}
