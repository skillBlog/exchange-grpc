package config

import (
	"os"
	"time"

	"github.com/exchange-grpc/shared/envparse"
)

const (
	defaultGRPCAddr           = ":50050"
	defaultJWTSecret          = "dev-exchange-secret"
	defaultAccessTokenTTL     = 15 * time.Minute
	defaultRefreshTokenTTL    = 7 * 24 * time.Hour
	defaultDatabaseURL        = "postgres://exchange:exchange@localhost:5432/userservice?sslmode=disable"
	defaultMigrationsDir      = "migrations"
	defaultLoginRateLimit     = 5
	defaultLoginRateWindow    = time.Minute
	defaultRedisURL           = "redis://localhost:6379/0"
	defaultRedisPoolSize      = 10
	defaultRedisMaxRetries    = 3
	defaultHealthCheckTimeout = 3 * time.Second
	defaultBcryptCost         = 12
	defaultMetricsAddr        = ":2112"
)

// Config содержит runtime-конфигурацию userservice.
type Config struct {
	GRPCAddr           string
	JWTSecret          string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	DatabaseURL        string
	MigrationsDir      string
	LoginRateLimit     int
	LoginRateWindow    time.Duration
	RedisURL           string
	RedisPoolSize      int
	RedisMaxRetries    int
	HealthCheckTimeout time.Duration
	LogLevelAddr       string
	MetricsAddr        string
	BcryptCost         int
}

// LoadConfig читает конфигурацию из переменных окружения.
func LoadConfig() (Config, error) {
	var env envparse.Loader
	cfg := Config{
		GRPCAddr:           env.String("USER_SERVICE_ADDR", defaultGRPCAddr),
		JWTSecret:          env.String("JWT_SECRET", defaultJWTSecret),
		AccessTokenTTL:     env.Duration("JWT_ACCESS_TTL", defaultAccessTokenTTL),
		RefreshTokenTTL:    env.Duration("JWT_REFRESH_TTL", defaultRefreshTokenTTL),
		DatabaseURL:        env.String("USER_DATABASE_URL", defaultDatabaseURL),
		MigrationsDir:      env.String("USER_MIGRATIONS_DIR", defaultMigrationsDir),
		LoginRateLimit:     env.Int("LOGIN_RATE_LIMIT", defaultLoginRateLimit),
		LoginRateWindow:    env.Duration("LOGIN_RATE_WINDOW", defaultLoginRateWindow),
		RedisURL:           env.String("REDIS_URL", defaultRedisURL),
		RedisPoolSize:      env.Int("REDIS_POOL_SIZE", defaultRedisPoolSize),
		RedisMaxRetries:    env.Int("REDIS_MAX_RETRIES", defaultRedisMaxRetries),
		HealthCheckTimeout: env.Duration("HEALTH_CHECK_TIMEOUT", defaultHealthCheckTimeout),
		LogLevelAddr:       os.Getenv("LOG_LEVEL_ADDR"),
		MetricsAddr:        env.String("METRICS_ADDR", defaultMetricsAddr),
		BcryptCost:         env.Int("BCRYPT_COST", defaultBcryptCost),
	}
	return cfg, env.Err()
}
