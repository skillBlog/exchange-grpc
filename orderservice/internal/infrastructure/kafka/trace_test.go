package kafka

import (
	"context"
	"testing"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestMessageFromEvent_carriesTraceIntoConsumerSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator())
	})

	relayCtx, relaySpan := tp.Tracer("test").Start(context.Background(), "outbox relay")
	msg, err := messageFromEvent(relayCtx, application.OutboxEvent{
		ID:          uuid.Must(uuid.NewV7()).String(),
		AggregateID: "11111111-1111-4111-8111-111111111111",
		EventType:   application.EventTypeOrderCreated,
		Payload:     []byte(`{}`),
	})
	relaySpan.End()
	if err != nil {
		t.Fatalf("messageFromEvent() error = %v", err)
	}

	if headerValue(msg.Headers, "event_id") == "" || headerValue(msg.Headers, "event_type") == "" {
		t.Fatalf("headers = %#v, want event_id and event_type", msg.Headers)
	}
	if headerValue(msg.Headers, "traceparent") == "" {
		t.Fatal("traceparent header is missing")
	}

	consumed := extractTrace(context.Background(), msg.Headers)
	_, consumeSpan := tp.Tracer("test").Start(consumed, "kafka.consume order.events")
	consumeSpan.End()

	var found bool
	parent := trace.SpanContextFromContext(relayCtx)
	for _, span := range exporter.GetSpans() {
		if span.Name != "kafka.consume order.events" {
			continue
		}
		found = true
		if span.SpanContext.TraceID() != parent.TraceID() {
			t.Fatalf("consumer trace = %s, want %s", span.SpanContext.TraceID(), parent.TraceID())
		}
		if span.Parent.SpanID() != parent.SpanID() {
			t.Fatalf("consumer parent = %s, want relay span %s", span.Parent.SpanID(), parent.SpanID())
		}
	}
	if !found {
		t.Fatal("consumer span was not recorded")
	}
}

func headerValue(headers []kafkago.Header, key string) string {
	for _, header := range headers {
		if header.Key == key {
			return string(header.Value)
		}
	}
	return ""
}
