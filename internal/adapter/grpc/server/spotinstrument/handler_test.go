package spotinstrument_test

import (
	"context"
	"net"
	"slices"
	"testing"
	"time"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	spotgrpc "github.com/exchange-grpc/internal/adapter/grpc/server/spotinstrument"
	"github.com/exchange-grpc/internal/adapter/repository/memory"
	"github.com/exchange-grpc/internal/platform/interceptor"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func TestServer_ViewMarkets_returnsActiveMarkets(t *testing.T) {
	client, cleanup := setupClient(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := client.ViewMarkets(ctx, &exchangev1.ViewMarketsRequest{
		UserRoles: []string{"trader"},
	})
	if err != nil {
		t.Fatalf("ViewMarkets() error = %v", err)
	}

	ids := make([]string, 0, len(resp.GetMarkets()))
	for _, market := range resp.GetMarkets() {
		if !market.GetEnabled() || market.GetDeletedAt() != nil {
			t.Fatalf("inactive market in response: %+v", market)
		}
		ids = append(ids, market.GetId())
	}

	want := []string{"BTC-USDT", "ETH-USDT", "BNB-USDT"}
	slices.Sort(ids)
	slices.Sort(want)
	if !slices.Equal(ids, want) {
		t.Fatalf("market ids = %v, want %v", ids, want)
	}
}

func setupClient(t *testing.T) (exchangev1.SpotInstrumentServiceClient, func()) {
	t.Helper()

	listener := bufconn.Listen(bufSize)
	marketRepo := memory.NewSeededMarketRepository()
	server := spotgrpc.NewServerFromRepository(marketRepo)

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
	)
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}

	cleanup := func() {
		conn.Close()
		grpcServer.Stop()
	}

	return exchangev1.NewSpotInstrumentServiceClient(conn), cleanup
}
