package config

import (
	"os"
	"strconv"
	"time"
)

const (
	defaultGRPCAddr         = ":50052"
	defaultSpotServiceHost  = "localhost:50051"
	defaultJWTSecret        = "dev-exchange-secret"
	defaultJWTTTL           = 24 * time.Hour
	defaultSpotGRPCTimeout  = 5 * time.Second
	defaultOrderHubBuffer   = 256
)

// Config содержит runtime-конфигурацию orderservice.
type Config struct {
	GRPCAddr           string
	SpotServiceHost    string
	JWTSecret          string
	JWTTTL             time.Duration
	SpotGRPCTimeout    time.Duration
	OrderHubBufferSize int
}

// LoadConfig читает конфигурацию из переменных окружения.
func LoadConfig() Config {
	return Config{
		GRPCAddr:           envOrDefault("ORDER_SERVICE_ADDR", defaultGRPCAddr),
		SpotServiceHost:    envOrDefault("SPOT_SERVICE_HOST", defaultSpotServiceHost),
		JWTSecret:          envOrDefault("JWT_SECRET", defaultJWTSecret),
		JWTTTL:             envDurationOrDefault("JWT_TTL", defaultJWTTTL),
		SpotGRPCTimeout:    envDurationOrDefault("SPOT_GRPC_TIMEOUT", defaultSpotGRPCTimeout),
		OrderHubBufferSize: envIntOrDefault("ORDER_HUB_BUFFER_SIZE", defaultOrderHubBuffer),
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
