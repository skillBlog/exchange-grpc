package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/exchange-grpc/shared/tracing"

// Start создаёт child span (под RPC-span от otelgrpc, если он есть).
func Start(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	opts := []trace.SpanStartOption{}
	if len(attrs) > 0 {
		opts = append(opts, trace.WithAttributes(attrs...))
	}
	return otel.Tracer(tracerName).Start(ctx, name, opts...)
}

// End завершает span и помечает ошибку, если err != nil.
// Передавайте указатель на named return: defer tracing.End(span, &err)
func End(span trace.Span, err *error) {
	if span == nil {
		return
	}
	if err != nil && *err != nil {
		span.RecordError(*err)
		span.SetStatus(codes.Error, (*err).Error())
	}
	span.End()
}

// Run создаёт child span вокруг fn.
func Run(ctx context.Context, name string, fn func(context.Context) error) (err error) {
	ctx, span := Start(ctx, name)
	defer End(span, &err)
	return fn(ctx)
}

// Attr — короткий alias для атрибутов span.
func Attr(key, value string) attribute.KeyValue {
	return attribute.String(key, value)
}
