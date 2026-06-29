package metrics_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	spotgrpc "github.com/exchange-grpc/internal/adapter/grpc/server/spotinstrument"
	"github.com/exchange-grpc/internal/adapter/repository/memory"
	"github.com/exchange-grpc/internal/platform/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func TestGRPCMetrics_ViewMarketsCounter(t *testing.T) {
	listener := bufconn.Listen(bufSize)
	marketRepo := memory.NewSeededMarketRepository()
	server := spotgrpc.NewServerFromRepository(marketRepo)

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(metrics.UnaryServerInterceptor()))
	exchangev1.RegisterSpotInstrumentServiceServer(grpcServer, server)
	metrics.RegisterGRPC(grpcServer)

	go func() {
		_ = grpcServer.Serve(listener)
	}()
	defer grpcServer.Stop()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	defer conn.Close()

	client := exchangev1.NewSpotInstrumentServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if _, err := client.ViewMarkets(ctx, &exchangev1.ViewMarketsRequest{}); err != nil {
		t.Fatalf("ViewMarkets() error = %v", err)
	}

	body := scrapeMetrics(t)
	if !strings.Contains(body, "grpc_server_handled_total") {
		t.Fatalf("metrics body missing grpc_server_handled_total:\n%s", body)
	}
	if !strings.Contains(body, "ViewMarkets") {
		t.Fatalf("metrics body missing ViewMarkets method:\n%s", body)
	}
}

func scrapeMetrics(t *testing.T) string {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr := "127.0.0.1:0"
	// эфемерный порт через listener
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	metricsAddr := listener.Addr().String()
	_ = listener.Close()

	go func() {
		_ = metrics.StartHTTPServer(ctx, metricsAddr)
	}()

	var body string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + metricsAddr + "/metrics")
		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		data, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			t.Fatalf("ReadAll() error = %v", readErr)
		}
		body = string(data)
		if strings.Contains(body, "grpc_server_handled_total") {
			cancel()
			return body
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatal("timed out waiting for metrics endpoint")
	return body
}
