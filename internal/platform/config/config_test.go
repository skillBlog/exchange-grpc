package config_test

import (
	"testing"
	"time"

	"github.com/exchange-grpc/internal/platform/config"
)

func TestLoad_defaults(t *testing.T) {
	t.Setenv("SPOT_INSTRUMENT_ADDR", "")
	t.Setenv("ORDER_ADDR", "")
	t.Setenv("ORDER_HOST", "")
	t.Setenv("SPOT_METRICS_ADDR", "")
	t.Setenv("ORDER_METRICS_ADDR", "")
	t.Setenv("SPOT_INSTRUMENT_HOST", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("REDIS_CACHE_TTL", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	cfg := config.Load()

	if cfg.SpotInstrumentAddr != ":50051" {
		t.Fatalf("SpotInstrumentAddr = %q, want :50051", cfg.SpotInstrumentAddr)
	}
	if cfg.OrderAddr != ":50052" {
		t.Fatalf("OrderAddr = %q, want :50052", cfg.OrderAddr)
	}
	if cfg.OrderHost != "localhost:50052" {
		t.Fatalf("OrderHost = %q, want localhost:50052", cfg.OrderHost)
	}
	if cfg.SpotMetricsAddr != ":9090" {
		t.Fatalf("SpotMetricsAddr = %q, want :9090", cfg.SpotMetricsAddr)
	}
	if cfg.OrderMetricsAddr != ":9091" {
		t.Fatalf("OrderMetricsAddr = %q, want :9091", cfg.OrderMetricsAddr)
	}
	if cfg.SpotInstrumentHost != "localhost:50051" {
		t.Fatalf("SpotInstrumentHost = %q, want localhost:50051", cfg.SpotInstrumentHost)
	}
	if cfg.RedisAddr != "" {
		t.Fatalf("RedisAddr = %q, want empty", cfg.RedisAddr)
	}
	if cfg.RedisCacheTTL != 30*time.Second {
		t.Fatalf("RedisCacheTTL = %v, want 30s", cfg.RedisCacheTTL)
	}
	if cfg.TracingOTLPEndpoint != "" {
		t.Fatalf("TracingOTLPEndpoint = %q, want empty", cfg.TracingOTLPEndpoint)
	}
}

func TestPort(t *testing.T) {
	port, err := config.Port(":50051")
	if err != nil {
		t.Fatalf("Port() error = %v", err)
	}
	if port != 50051 {
		t.Fatalf("port = %d, want 50051", port)
	}
}
