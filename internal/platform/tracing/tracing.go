package tracing

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

var (
	mu      sync.RWMutex
	enabled bool
)

// Enabled сообщает, активна ли трассировка OpenTelemetry.
func Enabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return enabled
}

// Init настраивает экспорт OTLP-трассировки в Jaeger или другой OTLP backend.
// Пустой endpoint отключает трассировку.
func Init(ctx context.Context, serviceName, otlpEndpoint string) (func(context.Context) error, error) {
	if strings.TrimSpace(otlpEndpoint) == "" {
		return func(context.Context) error { return nil }, nil
	}

	endpoint := strings.TrimSpace(otlpEndpoint)
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimSuffix(endpoint, "/")

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithURLPath("/v1/traces"),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otlp exporter: %w", err)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	mu.Lock()
	enabled = true
	mu.Unlock()

	return provider.Shutdown, nil
}
