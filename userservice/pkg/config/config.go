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
)

// Config содержит runtime-конфигурацию userservice.
type Config struct {
	GRPCAddr         string
	JWTSecret        string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	DatabaseURL      string
	MigrationsDir    string
	LoginRateLimit   int
	LoginRateWindow  time.Duration
}

// LoadConfig читает конфигурацию из переменных окружения.
func LoadConfig() Config {
	return Config{
		GRPCAddr:        envOrDefault("USER_SERVICE_ADDR", defaultGRPCAddr),
		JWTSecret:       envOrDefault("JWT_SECRET", defaultJWTSecret),
		AccessTokenTTL:  envDurationOrDefault("JWT_ACCESS_TTL", envDurationOrDefault("JWT_TTL", defaultAccessTokenTTL)),
		RefreshTokenTTL: envDurationOrDefault("JWT_REFRESH_TTL", defaultRefreshTokenTTL),
		DatabaseURL:     envOrDefault("USER_DATABASE_URL", defaultDatabaseURL),
		MigrationsDir:   envOrDefault("USER_MIGRATIONS_DIR", defaultMigrationsDir),
		LoginRateLimit:  envIntOrDefault("LOGIN_RATE_LIMIT", defaultLoginRateLimit),
		LoginRateWindow: envDurationOrDefault("LOGIN_RATE_WINDOW", defaultLoginRateWindow),
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
