package config

import "os"

const (
	defaultUserServiceHost  = "localhost:50050"
	defaultSpotServiceHost  = "localhost:50051"
	defaultOrderServiceHost = "localhost:50052"
)

// Config содержит адреса gRPC-сервисов для CLI-клиента.
type Config struct {
	UserServiceHost  string
	SpotServiceHost  string
	OrderServiceHost string
}

// LoadConfig читает конфигурацию из переменных окружения.
func LoadConfig() Config {
	return Config{
		UserServiceHost:  envOrDefault("USER_SERVICE_HOST", defaultUserServiceHost),
		SpotServiceHost:  envOrDefault("SPOT_SERVICE_HOST", defaultSpotServiceHost),
		OrderServiceHost: envOrDefault("ORDER_SERVICE_HOST", defaultOrderServiceHost),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
