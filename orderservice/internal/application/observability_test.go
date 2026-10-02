package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/memory"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/ratelimit"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestCreateOrder_rateLimitSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
	})

	uc := application.NewCreateOrder(
		memory.NewOrderRepository(),
		marketCheckerStub{},
		nil,
		nil,
		nil,
		ratelimit.NewCreateOrderLimiter(application.CreateOrderRateLimitConfig{
			GlobalLimit:  10,
			GlobalWindow: time.Minute,
			BasicLimit:   10,
			UserWindow:   time.Minute,
		}),
		nil,
	)
	if _, err := uc.Execute(context.Background(), application.CreateOrderInput{
		UserID:   "11111111-1111-1111-1111-111111111111",
		MarketID: "BTC-USDT",
		Side:     domain.OrderSideBuy,
		Quantity: mustDecimal(t, "0.1"),
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !hasSpan(exporter, "order.CreateOrder") {
		t.Fatal("expected order.CreateOrder span")
	}
	if !hasSpan(exporter, "order.CreateOrder.rateLimit") {
		t.Fatal("expected order.CreateOrder.rateLimit span")
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
