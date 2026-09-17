package application_test

import (
	"context"
	"testing"

	sharedgrpc "github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/spotservice/internal/application"
	"github.com/exchange-grpc/spotservice/internal/infrastructure/memory"
	"github.com/exchange-grpc/spotservice/internal/infrastructure/ratelimit"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestViewMarkets_rateLimitSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
	})

	uc := application.NewViewMarkets(
		memory.NewSeededMarketRepository(),
		ratelimit.NewViewMarketsLimiter(10, 0),
		nil,
	)
	if _, err := uc.Execute(context.Background(), application.ViewMarketsInput{
		UserID:    "user-1",
		UserRoles: []string{"trader"},
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !hasSpan(exporter, "spot.ViewMarkets") {
		t.Fatal("expected spot.ViewMarkets span")
	}
	if !hasSpan(exporter, "spot.viewMarketsRateLimit") {
		t.Fatal("expected spot.viewMarketsRateLimit span")
	}
}

func TestViewMarkets_auditIncludesRequestID(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	uc := application.NewViewMarkets(memory.NewSeededMarketRepository(), nil, zap.New(core))

	ctx := sharedgrpc.ContextWithRequestID(context.Background(), "req-spot-1")
	if _, err := uc.Execute(ctx, application.ViewMarketsInput{UserRoles: []string{"trader"}}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	entries := logs.FilterMessage("view markets").All()
	if len(entries) != 1 {
		t.Fatalf("audit logs = %d, want 1", len(entries))
	}
	fields := entries[0].ContextMap()
	if got, _ := fields["request_id"].(string); got != "req-spot-1" {
		t.Fatalf("request_id = %q", got)
	}
}

func hasSpan(exporter *tracetest.InMemoryExporter, name string) bool {
	for _, span := range exporter.GetSpans() {
		if span.Name == name {
			return true
		}
	}
	return false
}
