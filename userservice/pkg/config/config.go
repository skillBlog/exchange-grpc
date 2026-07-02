package config

import (
	"os"
	"time"
)

const (
	defaultGRPCAddr  = ":50050"
	defaultJWTSecret = "dev-exchange-secret"
	defaultJWTTTL    = 24 * time.Hour
)

// Config содержит runtime-конфигурацию userservice.
type Config struct {
	GRPCAddr  string
	JWTSecret string
	JWTTTL    time.Duration
}

// LoadConfig читает конфигурацию из переменных окружения.
func LoadConfig() Config {
	return Config{
		GRPCAddr:  envOrDefault("USER_SERVICE_ADDR", defaultGRPCAddr),
		JWTSecret: envOrDefault("JWT_SECRET", defaultJWTSecret),
		JWTTTL:    envDurationOrDefault("JWT_TTL", defaultJWTTTL),
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
