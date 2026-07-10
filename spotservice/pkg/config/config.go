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
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDurationOrDefault(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envIntOrDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
