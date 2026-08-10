package config

import (
	"os"
	"strconv"
	"time"
)

const (
	defaultGRPCAddr         = ":50050"
	defaultJWTSecret        = "dev-exchange-secret"
	defaultAccessTokenTTL   = 15 * time.Minute
	defaultRefreshTokenTTL  = 7 * 24 * time.Hour
	defaultDatabaseURL      = "postgres://exchange:exchange@localhost:5432/userservice?sslmode=disable"
	defaultMigrationsDir    = "migrations"
	defaultLoginRateLimit   = 5
	defaultLoginRateWindow  = time.Minute
	defaultRedisURL         = "redis://localhost:6379/0"
	defaultRedisPoolSize    = 10
	defaultRedisMaxRetries  = 3
	defaultHealthCheckTimeout = 3 * time.Second
)

// Config содержит runtime-конфигурацию userservice.
type Config struct {
	GRPCAddr            string
	JWTSecret           string
	AccessTokenTTL      time.Duration
	RefreshTokenTTL     time.Duration
	DatabaseURL         string
	MigrationsDir       string
	LoginRateLimit      int
	LoginRateWindow     time.Duration
	RedisURL            string
	RedisPoolSize       int
	RedisMaxRetries     int
	HealthCheckTimeout  time.Duration
	LogLevelAddr        string
}

// LoadConfig читает конфигурацию из переменных окружения.
func LoadConfig() Config {
	return Config{
		GRPCAddr:           envOrDefault("USER_SERVICE_ADDR", defaultGRPCAddr),
		JWTSecret:          envOrDefault("JWT_SECRET", defaultJWTSecret),
		AccessTokenTTL:     envDurationOrDefault("JWT_ACCESS_TTL", envDurationOrDefault("JWT_TTL", defaultAccessTokenTTL)),
		RefreshTokenTTL:    envDurationOrDefault("JWT_REFRESH_TTL", defaultRefreshTokenTTL),
		DatabaseURL:        envOrDefault("USER_DATABASE_URL", defaultDatabaseURL),
		MigrationsDir:      envOrDefault("USER_MIGRATIONS_DIR", defaultMigrationsDir),
		LoginRateLimit:     envIntOrDefault("LOGIN_RATE_LIMIT", defaultLoginRateLimit),
		LoginRateWindow:    envDurationOrDefault("LOGIN_RATE_WINDOW", defaultLoginRateWindow),
		RedisURL:           envOrDefault("REDIS_URL", defaultRedisURL),
		RedisPoolSize:      envIntOrDefault("REDIS_POOL_SIZE", defaultRedisPoolSize),
		RedisMaxRetries:    envIntOrDefault("REDIS_MAX_RETRIES", defaultRedisMaxRetries),
		HealthCheckTimeout: envDurationOrDefault("HEALTH_CHECK_TIMEOUT", defaultHealthCheckTimeout),
		LogLevelAddr:       os.Getenv("LOG_LEVEL_ADDR"),
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
