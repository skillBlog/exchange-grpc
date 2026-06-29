package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultSpotInstrumentAddr = ":50051"
	defaultOrderAddr          = ":50052"
	defaultOrderHost          = "localhost:50052"
	defaultSpotMetricsAddr    = ":9090"
	defaultOrderMetricsAddr   = ":9091"
	defaultSpotInstrumentHost = "localhost:50051"
	defaultRedisAddr          = ""
	defaultRedisCacheTTL      = 30 * time.Second
	defaultTracingOTLPEndpoint = ""
)

// Config содержит конфигурацию runtime, загружаемую из переменных окружения.
type Config struct {
	SpotInstrumentAddr string
	OrderAddr          string
	OrderHost          string
	SpotMetricsAddr    string
	OrderMetricsAddr   string
	SpotInstrumentHost string
	RedisAddr             string
	RedisCacheTTL         time.Duration
	TracingOTLPEndpoint   string
}

// Load читает конфигурацию из окружения со значениями по умолчанию.
func Load() Config {
	return Config{
		SpotInstrumentAddr: envOrDefault("SPOT_INSTRUMENT_ADDR", defaultSpotInstrumentAddr),
		OrderAddr:          envOrDefault("ORDER_ADDR", defaultOrderAddr),
		OrderHost:          envOrDefault("ORDER_HOST", defaultOrderHost),
		SpotMetricsAddr:    envOrDefault("SPOT_METRICS_ADDR", defaultSpotMetricsAddr),
		OrderMetricsAddr:   envOrDefault("ORDER_METRICS_ADDR", defaultOrderMetricsAddr),
		SpotInstrumentHost: envOrDefault("SPOT_INSTRUMENT_HOST", defaultSpotInstrumentHost),
		RedisAddr:            envOrDefault("REDIS_ADDR", defaultRedisAddr),
		RedisCacheTTL:        envDurationOrDefault("REDIS_CACHE_TTL", defaultRedisCacheTTL),
		TracingOTLPEndpoint:  envOrDefault("OTEL_EXPORTER_OTLP_ENDPOINT", defaultTracingOTLPEndpoint),
	}
}

// Port возвращает числовой порт из адреса вида ":50051".
func Port(addr string) (int, error) {
	_, portStr, err := splitHostPort(addr)
	if err != nil {
		return 0, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("invalid port %q: %w", portStr, err)
	}
	return port, nil
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

func splitHostPort(addr string) (host, port string, err error) {
	if addr == "" {
		return "", "", fmt.Errorf("empty address")
	}
	if addr[0] == ':' {
		return "", addr[1:], nil
	}
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i], addr[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("address %q has no port", addr)
}
