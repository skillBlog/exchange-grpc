package metrics_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/exchange-grpc/shared/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryServerInterceptor_recordsOKAndError(t *testing.T) {
	interceptor := metrics.UnaryServerInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/Login"}

	if _, err := interceptor(context.Background(), nil, info, func(context.Context, any) (any, error) {
		return "ok", nil
	}); err != nil {
		t.Fatalf("OK handler error = %v", err)
	}

	if _, err := interceptor(context.Background(), nil, info, func(context.Context, any) (any, error) {
		return nil, status.Error(codes.Unauthenticated, "nope")
	}); err == nil {
		t.Fatal("expected Unauthenticated")
	}

	body := scrape(t)
	for _, want := range []string{
		`grpc_server_started_total{grpc_method="Login",grpc_service="user.v1.UserService",grpc_type="unary"}`,
		`grpc_server_handled_total{grpc_code="OK",grpc_method="Login",grpc_service="user.v1.UserService",grpc_type="unary"}`,
		`grpc_server_handled_total{grpc_code="Unauthenticated",grpc_method="Login",grpc_service="user.v1.UserService",grpc_type="unary"}`,
		`grpc_server_handling_seconds_bucket{grpc_method="Login",grpc_service="user.v1.UserService",grpc_type="unary"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics body missing %q\n%s", want, body)
		}
	}
}

func TestStreamServerInterceptor_recordsServerStream(t *testing.T) {
	interceptor := metrics.StreamServerInterceptor()
	info := &grpc.StreamServerInfo{
		FullMethod:     "/order.v1.OrderService/StreamOrderUpdates",
		IsServerStream: true,
	}
	if err := interceptor(nil, nopStream{}, info, func(any, grpc.ServerStream) error {
		return nil
	}); err != nil {
		t.Fatalf("handler error = %v", err)
	}

	body := scrape(t)
	want := `grpc_server_handled_total{grpc_code="OK",grpc_method="StreamOrderUpdates",grpc_service="order.v1.OrderService",grpc_type="server_stream"}`
	if !strings.Contains(body, want) {
		t.Fatalf("metrics body missing %q\n%s", want, body)
	}
}

func TestServe_emptyAddrDisabled(t *testing.T) {
	if srv := metrics.Serve("  ", nil); srv != nil {
		t.Fatal("expected nil server for empty addr")
	}
	metrics.Shutdown(nil, nil)
}

func scrape(t *testing.T) string {
	t.Helper()
	rec := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(body)
}

type nopStream struct {
	grpc.ServerStream
}

func (nopStream) Context() context.Context { return context.Background() }
