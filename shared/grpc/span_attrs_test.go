package grpc_test

import (
	"context"
	"testing"

	sharedgrpc "github.com/exchange-grpc/shared/grpc"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/grpc"
)

func TestUnaryServerSpanAttrs_setsSafeAttributes(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
	})

	ctx, span := tp.Tracer("test").Start(context.Background(), "rpc")
	ctx = sharedgrpc.ContextWithRequestID(ctx, "req-1")
	ctx = sharedgrpc.ContextWithUserID(ctx, "user-1")

	_, err := sharedgrpc.UnaryServerSpanAttrs()(
		ctx,
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/order.v1.OrderService/CreateOrder"},
		func(context.Context, any) (any, error) { return nil, nil },
	)
	if err != nil {
		t.Fatalf("interceptor error = %v", err)
	}
	span.End()

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("spans = %d, want 1", len(ended))
	}
	attrs := map[string]string{}
	for _, attr := range ended[0].Attributes() {
		attrs[string(attr.Key)] = attr.Value.AsString()
	}
	if attrs["request_id"] != "req-1" {
		t.Fatalf("request_id = %q", attrs["request_id"])
	}
	if attrs["user_id"] != "user-1" {
		t.Fatalf("user_id = %q", attrs["user_id"])
	}
	if _, ok := attrs["email"]; ok {
		t.Fatal("email must not be a span attribute")
	}
}
