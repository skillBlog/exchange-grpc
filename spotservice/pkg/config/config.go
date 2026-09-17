package config

import (
	"os"
	"strconv"
	"time"
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
func LoadConfig() Config {
	return Config{
		GRPCAddr:              envOrDefault("SPOT_SERVICE_ADDR", defaultGRPCAddr),
		JWTSecret:             envOrDefault("JWT_SECRET", defaultJWTSecret),
		AccessTokenTTL:        envDurationOrDefault("JWT_ACCESS_TTL", envDurationOrDefault("JWT_TTL", defaultAccessTokenTTL)),
		DatabaseURL:           envOrDefault("SPOT_DATABASE_URL", defaultDatabaseURL),
		MigrationsDir:         envOrDefault("SPOT_MIGRATIONS_DIR", defaultMigrationsDir),
		MarketCacheTTL:        envDurationOrDefault("MARKET_CACHE_TTL", defaultMarketCacheTTL),
		ViewMarketsRateLimit:  envIntOrDefault("VIEW_MARKETS_RATE_LIMIT", defaultViewMarketsRateLimit),
		ViewMarketsRateWindow: envDurationOrDefault("VIEW_MARKETS_RATE_WINDOW", defaultViewMarketsRateWindow),
		HealthCheckTimeout:    envDurationOrDefault("HEALTH_CHECK_TIMEOUT", defaultHealthCheckTimeout),
		LogLevelAddr:          os.Getenv("LOG_LEVEL_ADDR"),
		RedisURL:              envOrDefault("REDIS_URL", defaultRedisURL),
		RedisPoolSize:         envIntOrDefault("REDIS_POOL_SIZE", defaultRedisPoolSize),
		RedisMaxRetries:       envIntOrDefault("REDIS_MAX_RETRIES", defaultRedisMaxRetries),
		MetricsAddr:           envOrDefault("METRICS_ADDR", defaultMetricsAddr),
	}
}

func envOrDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func envDurationOrDefault(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envIntOrDefault(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
