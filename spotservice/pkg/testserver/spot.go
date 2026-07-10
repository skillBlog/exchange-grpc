package testserver

import (
	"context"
	"net"
	"testing"
	"time"

	spotv1 "github.com/exchange-grpc/proto/pb/spot/v1"
	grpcserver "github.com/exchange-grpc/spotservice/internal/interfaces/grpcserver"
	"github.com/exchange-grpc/spotservice/internal/infrastructure/memory"
	"github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/shared/sessionvalidation"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// Spot запускает spotservice in-process через bufconn.
type Spot struct {
	Client spotv1.SpotServiceClient
	Conn   *googlegrpc.ClientConn
	Server *googlegrpc.Server
}

// NewSpot поднимает SpotService с JWT auth для интеграционных тестов.
func NewSpot(t *testing.T, tokens *sessionvalidation.TokenService) *Spot {
	t.Helper()

	unary := grpc.ChainUnaryServer(
		grpc.UnaryServerRequestID,
		grpc.NewUnaryServerJWTAuth(tokens),
	)

	listener := bufconn.Listen(bufSize)
	repo := memory.NewSeededMarketRepository()
	server := grpcserver.NewServerFromRepository(repo, nil)
	grpcServer := googlegrpc.NewServer(googlegrpc.UnaryInterceptor(unary))
	spotv1.RegisterSpotServiceServer(grpcServer, server)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	conn, err := googlegrpc.NewClient(
		"passthrough:///spot",
		googlegrpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		googlegrpc.WithTransportCredentials(insecure.NewCredentials()),
		googlegrpc.WithUnaryInterceptor(grpc.ChainUnaryClient(
			grpc.UnaryClientRequestID,
			grpc.UnaryClientForwardAuthorization,
		)),
	)
	if err != nil {
		t.Fatalf("spot grpc.NewClient() error = %v", err)
	}

	t.Cleanup(func() {
		conn.Close()
		grpcServer.Stop()
	})

	return &Spot{
		Client: spotv1.NewSpotServiceClient(conn),
		Conn:   conn,
		Server: grpcServer,
	}
}

// TestTokenService создаёт TokenService с фиксированным секретом для тестов.
func TestTokenService() *sessionvalidation.TokenService {
	svc, err := sessionvalidation.NewTokenService("integration-test-secret", time.Hour)
	if err != nil {
		panic(err)
	}
	return svc
}
