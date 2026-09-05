package logger_test

import (
	"context"
	"testing"

	"github.com/exchange-grpc/shared/logger"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestWithTrace_addsTraceAndSpanIDs(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
	})

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	core, logs := observer.New(zapcore.InfoLevel)
	logger.WithTrace(ctx, zap.New(core)).Info("hello")

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	fields := entries[0].ContextMap()
	if fields["trace_id"] != span.SpanContext().TraceID().String() {
		t.Fatalf("trace_id = %v", fields["trace_id"])
	}
	if fields["span_id"] != span.SpanContext().SpanID().String() {
		t.Fatalf("span_id = %v", fields["span_id"])
	}
}

func TestWithTrace_noopWithoutSpan(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger.WithTrace(context.Background(), zap.New(core)).Info("hello")
	if logs.Len() != 1 {
		t.Fatalf("entries = %d, want 1", logs.Len())
	}
	if _, ok := logs.All()[0].ContextMap()["trace_id"]; ok {
		t.Fatal("did not expect trace_id without span")
	}
}
