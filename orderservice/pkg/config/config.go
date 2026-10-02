package config

import (
	"os"
	"strings"
	"time"

	"github.com/exchange-grpc/shared/envparse"
)

const (
	defaultGRPCAddr               = ":50052"
	defaultSpotServiceHost        = "localhost:50051"
	defaultJWTSecret              = "dev-exchange-secret"
	defaultAccessTokenTTL         = 15 * time.Minute
	defaultSpotGRPCTimeout        = 5 * time.Second
	defaultOrderHubBuffer         = 256
	defaultOrderHubPublishTimeout = 100 * time.Millisecond
	defaultIdempotencyTTL         = 24 * time.Hour
	defaultHealthCheckTimeout     = 3 * time.Second
	defaultDatabaseURL            = "postgres://exchange:exchange@localhost:5432/orderservice?sslmode=disable"
	defaultMigrationsDir          = "migrations"
	defaultRedisURL               = "redis://localhost:6379/0"
	defaultRedisPoolSize          = 10
	defaultRedisMaxRetries        = 3
	defaultMetricsAddr            = ":2112"

	defaultCreateOrderGlobalLimit  = 20000
	defaultCreateOrderBasicLimit   = 10
	defaultCreateOrderPremiumLimit = 100
	defaultCreateOrderAdminLimit   = 1000
	defaultCreateOrderRateWindow   = time.Minute
	defaultKafkaOrderTopic         = "order.events"
	defaultKafkaProduceTimeout     = 5 * time.Second
	defaultKafkaCommandTopic       = "order.commands"
	defaultKafkaCommandDLQTopic    = "order.commands.dlq"
	defaultKafkaCommandGroup       = "orderservice-commands"
	defaultKafkaCommandMaxAttempts = 5
)

// Config содержит runtime-конфигурацию orderservice.
type Config struct {
	GRPCAddr               string
	SpotServiceHost        string
	JWTSecret              string
	AccessTokenTTL         time.Duration
	SpotGRPCTimeout        time.Duration
	OrderHubBufferSize     int
	OrderHubPublishTimeout time.Duration
	IdempotencyTTL         time.Duration
	HealthCheckTimeout     time.Duration
	DatabaseURL            string
	MigrationsDir          string
	RedisURL               string
	RedisPoolSize          int
	RedisMaxRetries        int
	LogLevelAddr           string
	MetricsAddr            string
	CreateOrderRateLimit   CreateOrderRateLimitConfig
	KafkaBrokers           []string
	KafkaOrderTopic        string
	KafkaProduceTimeout    time.Duration
	KafkaCommandTopic      string
	KafkaCommandDLQTopic   string
	KafkaCommandGroup      string
	KafkaCommandMaxAttempts int
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
func LoadConfig() (Config, error) {
	var env envparse.Loader
	cfg := Config{
		GRPCAddr:               env.String("ORDER_SERVICE_ADDR", defaultGRPCAddr),
		SpotServiceHost:        env.String("SPOT_SERVICE_HOST", defaultSpotServiceHost),
		JWTSecret:              env.String("JWT_SECRET", defaultJWTSecret),
		AccessTokenTTL:         env.Duration("JWT_ACCESS_TTL", defaultAccessTokenTTL),
		SpotGRPCTimeout:        env.Duration("SPOT_GRPC_TIMEOUT", defaultSpotGRPCTimeout),
		OrderHubBufferSize:     env.Int("ORDER_HUB_BUFFER_SIZE", defaultOrderHubBuffer),
		OrderHubPublishTimeout: env.Duration("ORDER_HUB_PUBLISH_TIMEOUT", defaultOrderHubPublishTimeout),
		IdempotencyTTL:         env.Duration("IDEMPOTENCY_TTL", defaultIdempotencyTTL),
		HealthCheckTimeout:     env.Duration("HEALTH_CHECK_TIMEOUT", defaultHealthCheckTimeout),
		DatabaseURL:            env.String("ORDER_DATABASE_URL", defaultDatabaseURL),
		MigrationsDir:          env.String("ORDER_MIGRATIONS_DIR", defaultMigrationsDir),
		RedisURL:               env.String("REDIS_URL", defaultRedisURL),
		RedisPoolSize:          env.Int("REDIS_POOL_SIZE", defaultRedisPoolSize),
		RedisMaxRetries:        env.Int("REDIS_MAX_RETRIES", defaultRedisMaxRetries),
		LogLevelAddr:           os.Getenv("LOG_LEVEL_ADDR"),
		MetricsAddr:            env.String("METRICS_ADDR", defaultMetricsAddr),
		CreateOrderRateLimit: CreateOrderRateLimitConfig{
			GlobalLimit:  env.Int("CREATE_ORDER_GLOBAL_RATE_LIMIT", defaultCreateOrderGlobalLimit),
			GlobalWindow: env.Duration("CREATE_ORDER_GLOBAL_RATE_WINDOW", defaultCreateOrderRateWindow),
			BasicLimit:   env.Int("CREATE_ORDER_RATE_LIMIT_USER", defaultCreateOrderBasicLimit),
			PremiumLimit: env.Int("CREATE_ORDER_RATE_LIMIT_TRADER", defaultCreateOrderPremiumLimit),
			AdminLimit:   env.Int("CREATE_ORDER_RATE_LIMIT_ADMIN", defaultCreateOrderAdminLimit),
			UserWindow:   env.Duration("CREATE_ORDER_RATE_WINDOW", defaultCreateOrderRateWindow),
		},
		KafkaBrokers:            splitCSV(env.String("KAFKA_BROKERS", "")),
		KafkaOrderTopic:         env.String("KAFKA_ORDER_TOPIC", defaultKafkaOrderTopic),
		KafkaProduceTimeout:     env.Duration("KAFKA_PRODUCE_TIMEOUT", defaultKafkaProduceTimeout),
		KafkaCommandTopic:       env.String("KAFKA_COMMAND_TOPIC", defaultKafkaCommandTopic),
		KafkaCommandDLQTopic:    env.String("KAFKA_COMMAND_DLQ_TOPIC", defaultKafkaCommandDLQTopic),
		KafkaCommandGroup:       env.String("KAFKA_COMMAND_GROUP", defaultKafkaCommandGroup),
		KafkaCommandMaxAttempts: env.Int("KAFKA_COMMAND_MAX_ATTEMPTS", defaultKafkaCommandMaxAttempts),
	}
	return cfg, env.Err()
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}
