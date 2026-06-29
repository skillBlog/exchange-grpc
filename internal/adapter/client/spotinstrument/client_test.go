package spotinstrument_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	spotclient "github.com/exchange-grpc/internal/adapter/client/spotinstrument"
	spotgrpc "github.com/exchange-grpc/internal/adapter/grpc/server/spotinstrument"
	"github.com/exchange-grpc/internal/adapter/repository/memory"
	"github.com/exchange-grpc/internal/domain"
	"github.com/exchange-grpc/internal/platform/interceptor"
	"go.uber.org/zap"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func TestClient_EnsureMarketAvailable_activeMarket(t *testing.T) {
	client, cleanup := setupClient(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.EnsureMarketAvailable(ctx, "BTC-USDT", nil); err != nil {
		t.Fatalf("EnsureMarketAvailable() error = %v", err)
	}
}

func TestClient_EnsureMarketAvailable_inactiveMarket(t *testing.T) {
	client, cleanup := setupClient(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := client.EnsureMarketAvailable(ctx, "SOL-USDT", nil)
	if !errors.Is(err, domain.ErrMarketInactive) {
		t.Fatalf("error = %v, want ErrMarketInactive", err)
	}
}

func TestClient_EnsureMarketAvailable_deletedMarket(t *testing.T) {
	client, cleanup := setupClient(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := client.EnsureMarketAvailable(ctx, "XRP-USDT", nil)
	if !errors.Is(err, domain.ErrMarketInactive) {
		t.Fatalf("error = %v, want ErrMarketInactive", err)
	}
}

func TestClient_EnsureMarketAvailable_unknownMarket(t *testing.T) {
	client, cleanup := setupClient(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := client.EnsureMarketAvailable(ctx, "DOGE-USDT", nil)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestClient_EnsureMarketAvailable_forbiddenMarket(t *testing.T) {
	client, cleanup := setupClient(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := client.EnsureMarketAvailable(ctx, "BNB-USDT", nil)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}

	if err := client.EnsureMarketAvailable(ctx, "BNB-USDT", []string{"trader"}); err != nil {
		t.Fatalf("EnsureMarketAvailable(trader) error = %v", err)
	}
}

func TestClient_propagatesRequestID(t *testing.T) {
	var captured metadata.MD
	listener := bufconn.Listen(bufSize)

	marketRepo := memory.NewSeededMarketRepository()
	server := spotgrpc.NewServerFromRepository(marketRepo)

	log := zap.NewNop()
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(interceptor.ChainUnaryServer(
		interceptor.UnaryServerRequestID,
		interceptor.UnaryServerLogger(log),
		interceptor.UnaryServerPanicRecovery(log),
		func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				captured = md.Copy()
			}
			return handler(ctx, req)
		},
	)))
	exchangev1.RegisterSpotInstrumentServiceServer(grpcServer, server)

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
		grpc.WithUnaryInterceptor(interceptor.UnaryClientRequestID),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	defer conn.Close()

	client := spotclient.New(conn)
	ctx := interceptor.ContextWithRequestID(context.Background(), "req-trace-42")

	if err := client.EnsureMarketAvailable(ctx, "BTC-USDT", nil); err != nil {
		t.Fatalf("EnsureMarketAvailable() error = %v", err)
	}

	values := captured.Get(interceptor.MetadataKey)
	if len(values) != 1 || values[0] != "req-trace-42" {
		t.Fatalf("incoming request_id = %v, want req-trace-42", values)
	}
}

func setupClient(t *testing.T) (*spotclient.Client, func()) {
	t.Helper()

	listener := bufconn.Listen(bufSize)
	server := spotgrpc.NewServerFromRepository(memory.NewSeededMarketRepository())

	log := zap.NewNop()
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(interceptor.ChainUnaryServer(
		interceptor.UnaryServerRequestID,
		interceptor.UnaryServerLogger(log),
		interceptor.UnaryServerPanicRecovery(log),
	)))
	exchangev1.RegisterSpotInstrumentServiceServer(grpcServer, server)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptor.UnaryClientRequestID),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}

	cleanup := func() {
		conn.Close()
		grpcServer.Stop()
	}

	return spotclient.New(conn), cleanup
}
