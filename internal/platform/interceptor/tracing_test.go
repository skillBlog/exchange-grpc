package interceptor_test

import (
	"context"
	"testing"

	"github.com/exchange-grpc/internal/platform/interceptor"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/grpc"
)

func TestUnaryServerSpanRequestID_setsAttribute(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	otel.SetTracerProvider(provider)

	tracer := provider.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "rpc")
	defer span.End()

	const requestID = "trace-req-1"
	ctx = interceptor.ContextWithRequestID(ctx, requestID)

	handler := func(ctx context.Context, _ any) (any, error) {
		return "ok", nil
	}

	_, err := interceptor.UnaryServerSpanRequestID(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: "/exchange.v1.OrderService/CreateOrder",
	}, handler)
	if err != nil {
		t.Fatalf("UnaryServerSpanRequestID() error = %v", err)
	}

	span.End()

	recorded := sr.Ended()
	if len(recorded) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(recorded))
	}

	attrs := recorded[0].Attributes()
	if !hasAttribute(attrs, "request_id", requestID) {
		t.Fatalf("request_id attribute missing: %+v", attrs)
	}
}

func hasAttribute(attrs []attribute.KeyValue, key, want string) bool {
	for _, attr := range attrs {
		if string(attr.Key) == key && attr.Value.AsString() == want {
			return true
		}
	}
	return false
}
