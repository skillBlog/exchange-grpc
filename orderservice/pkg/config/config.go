package config

import (
	"os"
	"strconv"
	"time"
)

const (
	defaultGRPCAddr        = ":50052"
	defaultSpotServiceHost = "localhost:50051"
	defaultJWTSecret       = "dev-exchange-secret"
	defaultAccessTokenTTL  = 15 * time.Minute
	defaultSpotGRPCTimeout = 5 * time.Second
	defaultOrderHubBuffer  = 256
	defaultDatabaseURL     = "postgres://exchange:exchange@localhost:5432/orderservice?sslmode=disable"
	defaultMigrationsDir   = "migrations"
	defaultRedisURL        = "redis://localhost:6379/0"

	defaultCreateOrderGlobalLimit  = 20000
	defaultCreateOrderBasicLimit   = 10
	defaultCreateOrderPremiumLimit = 100
	defaultCreateOrderAdminLimit   = 1000
	defaultCreateOrderRateWindow   = time.Minute
)

// Config содержит runtime-конфигурацию orderservice.
type Config struct {
	GRPCAddr           string
	SpotServiceHost    string
	JWTSecret          string
	AccessTokenTTL     time.Duration
	SpotGRPCTimeout    time.Duration
	OrderHubBufferSize int
	DatabaseURL        string
	MigrationsDir      string
	RedisURL           string
	CreateOrderRateLimit CreateOrderRateLimitConfig
}

// CreateOrderRateLimitConfig — лимиты CreateOrder из ENV.
type CreateOrderRateLimitConfig struct {
	GlobalLimit  int
	GlobalWindow time.Duration
	BasicLimit   int
	PremiumLimit int
	AdminLimit   int
	UserWindow   time.Duration
}

// LoadConfig читает конфигурацию из переменных окружения.
func LoadConfig() Config {
	return Config{
		GRPCAddr:           envOrDefault("ORDER_SERVICE_ADDR", defaultGRPCAddr),
		SpotServiceHost:    envOrDefault("SPOT_SERVICE_HOST", defaultSpotServiceHost),
		JWTSecret:          envOrDefault("JWT_SECRET", defaultJWTSecret),
		AccessTokenTTL:     envDurationOrDefault("JWT_ACCESS_TTL", envDurationOrDefault("JWT_TTL", defaultAccessTokenTTL)),
		SpotGRPCTimeout:    envDurationOrDefault("SPOT_GRPC_TIMEOUT", defaultSpotGRPCTimeout),
		OrderHubBufferSize: envIntOrDefault("ORDER_HUB_BUFFER_SIZE", defaultOrderHubBuffer),
		DatabaseURL:        envOrDefault("ORDER_DATABASE_URL", defaultDatabaseURL),
		MigrationsDir:      envOrDefault("ORDER_MIGRATIONS_DIR", defaultMigrationsDir),
		RedisURL:           envOrDefault("REDIS_URL", defaultRedisURL),
		CreateOrderRateLimit: CreateOrderRateLimitConfig{
			GlobalLimit:  envIntOrDefault("CREATE_ORDER_GLOBAL_RATE_LIMIT", defaultCreateOrderGlobalLimit),
			GlobalWindow: envDurationOrDefault("CREATE_ORDER_GLOBAL_RATE_WINDOW", defaultCreateOrderRateWindow),
			BasicLimit:   envIntOrDefault("CREATE_ORDER_RATE_LIMIT_USER", defaultCreateOrderBasicLimit),
			PremiumLimit: envIntOrDefault("CREATE_ORDER_RATE_LIMIT_TRADER", defaultCreateOrderPremiumLimit),
			AdminLimit:   envIntOrDefault("CREATE_ORDER_RATE_LIMIT_ADMIN", defaultCreateOrderAdminLimit),
			UserWindow:   envDurationOrDefault("CREATE_ORDER_RATE_WINDOW", defaultCreateOrderRateWindow),
		},
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
