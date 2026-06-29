package integration

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	spotclient "github.com/exchange-grpc/internal/adapter/client/spotinstrument"
	ordergrpc "github.com/exchange-grpc/internal/adapter/grpc/server/order"
	spotgrpc "github.com/exchange-grpc/internal/adapter/grpc/server/spotinstrument"
	"github.com/exchange-grpc/internal/adapter/repository/memory"
	"github.com/exchange-grpc/internal/platform/interceptor"
	"github.com/exchange-grpc/internal/platform/metrics"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// Suite запускает SpotInstrument и Order gRPC-сервисы in-process для интеграционных тестов.
type Suite struct {
	SpotClient    exchangev1.SpotInstrumentServiceClient
	OrderClient   exchangev1.OrderServiceClient
	OrderServices ordergrpc.Services
	Logs          *observer.ObservedLogs

	spotGRPC  *grpc.Server
	orderGRPC *grpc.Server
	spotConn  *grpc.ClientConn
	orderConn *grpc.ClientConn
}

// NewSuite подключает оба сервиса с интерсепторами, как в production.
// При observeLogs=true логи сервера сохраняются для проверок в тестах.
func NewSuite(t *testing.T, observeLogs bool) *Suite {
	t.Helper()

	var (
		log  *zap.Logger
		logs *observer.ObservedLogs
	)
	if observeLogs {
		core, observed := observer.New(zap.InfoLevel)
		logs = observed
		log = zap.New(core)
	} else {
		log = zap.NewNop()
	}

	unary := interceptor.ChainUnaryServer(
		interceptor.UnaryServerRequestID,
		interceptor.UnaryServerLogger(log),
		interceptor.UnaryServerPanicRecovery(log),
		metrics.UnaryServerInterceptor(),
	)

	spotListener := bufconn.Listen(bufSize)
	spotRepo := memory.NewSeededMarketRepository()
	spotServer := spotgrpc.NewServerFromRepository(spotRepo)
	spotGRPC := grpc.NewServer(grpc.UnaryInterceptor(unary))
	exchangev1.RegisterSpotInstrumentServiceServer(spotGRPC, spotServer)
	metrics.RegisterGRPC(spotGRPC)

	go func() {
		_ = spotGRPC.Serve(spotListener)
	}()

	spotConn, err := grpc.NewClient(
		"passthrough:///spot",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return spotListener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptor.UnaryClientRequestID),
	)
	if err != nil {
		t.Fatalf("spot grpc.NewClient() error = %v", err)
	}

	marketClient := spotclient.New(spotConn)
	orderRepo := memory.NewOrderRepository()
	orderServices := ordergrpc.NewServices(orderRepo, marketClient)
	orderServer := ordergrpc.NewServer(orderServices)

	orderListener := bufconn.Listen(bufSize)
	orderGRPC := grpc.NewServer(grpc.UnaryInterceptor(unary))
	exchangev1.RegisterOrderServiceServer(orderGRPC, orderServer)
	metrics.RegisterGRPC(orderGRPC)

	go func() {
		_ = orderGRPC.Serve(orderListener)
	}()

	orderConn, err := grpc.NewClient(
		"passthrough:///order",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return orderListener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptor.UnaryClientRequestID),
	)
	if err != nil {
		t.Fatalf("order grpc.NewClient() error = %v", err)
	}

	t.Cleanup(func() {
		orderConn.Close()
		spotConn.Close()
		orderGRPC.Stop()
		spotGRPC.Stop()
	})

	return &Suite{
		SpotClient:    exchangev1.NewSpotInstrumentServiceClient(spotConn),
		OrderClient:   exchangev1.NewOrderServiceClient(orderConn),
		OrderServices: orderServices,
		Logs:          logs,
		spotGRPC:    spotGRPC,
		orderGRPC:   orderGRPC,
		spotConn:    spotConn,
		orderConn:   orderConn,
	}
}

// ScrapeMetrics возвращает тело exposition Prometheus из реестра по умолчанию.
func ScrapeMetrics(t *testing.T) string {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	go func() {
		_ = metrics.StartHTTPServer(ctx, addr)
	}()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + addr + "/metrics")
		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			t.Fatalf("ReadAll() error = %v", readErr)
		}
		if len(body) > 0 {
			cancel()
			return string(body)
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatal("timed out waiting for /metrics")
	return ""
}

// ContainsMetric проверяет, что текст exposition содержит метрику и фрагмент label.
func ContainsMetric(body, metric, fragment string) bool {
	if !strings.Contains(body, metric) {
		return false
	}
	return fragment == "" || strings.Contains(body, fragment)
}
