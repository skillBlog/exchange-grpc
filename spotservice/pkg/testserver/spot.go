package testserver

import (
	"context"
	"net"
	"testing"
	"time"

	spotv1 "github.com/exchange-grpc/proto/pb/spot/v1"
	"github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/shared/sessionvalidation"
	"github.com/exchange-grpc/spotservice/internal/infrastructure/memory"
	grpcserver "github.com/exchange-grpc/spotservice/internal/interfaces/grpcserver"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// Spot запускает spotservice in-process через bufconn.
type Spot struct {
	Client       spotv1.SpotServiceClient
	HealthClient grpc_health_v1.HealthClient
	Conn         *googlegrpc.ClientConn
	Server       *googlegrpc.Server
}

// NewSpot поднимает SpotService с JWT auth для интеграционных тестов.
func NewSpot(t *testing.T, tokens *sessionvalidation.TokenService) *Spot {
	t.Helper()

	validator, err := grpc.NewProtoValidator()
	if err != nil {
		t.Fatalf("NewProtoValidator() error = %v", err)
	}

	unary := grpc.UnaryServerInterceptors(nil, validator, tokens, grpc_health_v1.Health_Check_FullMethodName)
	stream := grpc.StreamServerInterceptors(nil, validator, tokens, grpc_health_v1.Health_Watch_FullMethodName)

	listener := bufconn.Listen(bufSize)
	repo := memory.NewSeededMarketRepository()
	server := grpcserver.NewServerFromRepository(repo, nil, nil)
	grpcServer := googlegrpc.NewServer(
		googlegrpc.UnaryInterceptor(unary),
		googlegrpc.StreamInterceptor(stream),
	)
	spotv1.RegisterSpotServiceServer(grpcServer, server)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus(spotv1.SpotService_ServiceDesc.ServiceName, grpc_health_v1.HealthCheckResponse_SERVING)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	conn, err := googlegrpc.NewClient(
		"passthrough:///spot",
		googlegrpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		googlegrpc.WithTransportCredentials(insecure.NewCredentials()),
		googlegrpc.WithChainUnaryInterceptor(
			grpc.UnaryClientRequestID,
			grpc.UnaryClientForwardAuthorization,
		),
	)
	if err != nil {
		t.Fatalf("spot grpc.NewClient() error = %v", err)
	}

	t.Cleanup(func() {
		conn.Close()
		grpcServer.Stop()
	})

	return &Spot{
		Client:       spotv1.NewSpotServiceClient(conn),
		HealthClient: grpc_health_v1.NewHealthClient(conn),
		Conn:         conn,
		Server:       grpcServer,
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
