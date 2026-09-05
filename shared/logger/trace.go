package logger

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// WithTrace возвращает логгер с trace_id и span_id из контекста, если спан валиден.
func WithTrace(ctx context.Context, log *zap.Logger) *zap.Logger {
	if log == nil {
		log = zap.NewNop()
	}
	if ctx == nil {
		return log
	}
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return log
	}
	return log.With(
		zap.String("trace_id", sc.TraceID().String()),
		zap.String("span_id", sc.SpanID().String()),
	)
}
