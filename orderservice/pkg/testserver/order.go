package testserver

import (
	"context"
	"net"
	"testing"
	"time"

	orderv1 "github.com/exchange-grpc/proto/pb/order/v1"
	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/memory"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/ratelimit"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/spotclient"
	grpcserver "github.com/exchange-grpc/orderservice/internal/interfaces/grpcserver"
	"github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/shared/sessionvalidation"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// Services — публичный alias для доступа к use case'ам в интеграционных тестах.
type Services = grpcserver.Services

// UpdateOrderStatusInput — параметры смены статуса ордера в тестах.
type UpdateOrderStatusInput = application.UpdateOrderStatusInput

// OrderStatus — доменный статус ордера для тестов.
type OrderStatus = domain.OrderStatus

const (
	OrderStatusFilled = domain.OrderStatusFilled
)

// UpdateOrderStatus меняет статус ордера в интеграционных тестах.
func UpdateOrderStatus(ctx context.Context, services Services, input UpdateOrderStatusInput) error {
	return services.UpdateOrderStatus.Execute(ctx, input)
}

// Order запускает orderservice in-process через bufconn.
type Order struct {
	Client   orderv1.OrderServiceClient
	Services grpcserver.Services
	Conn     *googlegrpc.ClientConn
	Server   *googlegrpc.Server
}

// NewOrder поднимает OrderService, подключённый к spot через bufconn.
func NewOrder(t *testing.T, spotConn *googlegrpc.ClientConn, tokens *sessionvalidation.TokenService) *Order {
	t.Helper()

	unary := grpc.ChainUnaryServer(
		grpc.UnaryServerRequestID,
		grpc.NewUnaryServerJWTAuth(tokens),
	)

	marketClient := spotclient.New(spotConn, 0)
	orderRepo := memory.NewOrderRepository()
	idempotencyStore := memory.NewIdempotencyStore()
	createOrderLimiter := ratelimit.NewCreateOrderLimiter(application.CreateOrderRateLimitConfig{
		GlobalLimit:  20000,
		GlobalWindow: time.Minute,
		BasicLimit:   1000,
		PremiumLimit: 1000,
		AdminLimit:   1000,
		UserWindow:   time.Minute,
	})
	orderServices := grpcserver.NewServices(orderRepo, idempotencyStore, marketClient, createOrderLimiter, 256, nil)
	orderServer := grpcserver.NewServer(orderServices)

	listener := bufconn.Listen(bufSize)
	grpcServer := googlegrpc.NewServer(
		googlegrpc.UnaryInterceptor(unary),
		googlegrpc.StreamInterceptor(grpc.ChainStreamServer(
			grpc.StreamServerRequestID,
			grpc.NewStreamServerJWTAuth(tokens),
		)),
	)
	orderv1.RegisterOrderServiceServer(grpcServer, orderServer)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	conn, err := googlegrpc.NewClient(
		"passthrough:///order",
		googlegrpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		googlegrpc.WithTransportCredentials(insecure.NewCredentials()),
		googlegrpc.WithUnaryInterceptor(grpc.UnaryClientRequestID),
	)
	if err != nil {
		t.Fatalf("order grpc.NewClient() error = %v", err)
	}

	t.Cleanup(func() {
		conn.Close()
		grpcServer.Stop()
	})

	return &Order{
		Client:   orderv1.NewOrderServiceClient(conn),
		Services: orderServices,
		Conn:     conn,
		Server:   grpcServer,
	}
}
