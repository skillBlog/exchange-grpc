package tracing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/exchange-grpc/shared/tracing"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel"
)

func TestStartEnd_recordsError(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
	})

	var err error
	ctx, span := tracing.Start(context.Background(), "test.op")
	err = errors.New("boom")
	tracing.End(span, &err)
	_ = ctx

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("spans = %d, want 1", len(spans))
	}
	if spans[0].Status.Code != codes.Error {
		t.Fatalf("status = %v, want Error", spans[0].Status.Code)
	}
}

func TestRun_recordsError(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
	})

	boom := errors.New("boom")
	err := tracing.Run(context.Background(), "test.run", func(context.Context) error {
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("Run() error = %v, want boom", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("spans = %d, want 1", len(spans))
	}
	if spans[0].Name != "test.run" {
		t.Fatalf("name = %q, want test.run", spans[0].Name)
	}
	if spans[0].Status.Code != codes.Error {
		t.Fatalf("status = %v, want Error", spans[0].Status.Code)
	}
}
