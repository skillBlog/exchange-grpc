package order_test

import (
	"context"
	"net"
	"testing"
	"time"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	spotclient "github.com/exchange-grpc/internal/adapter/client/spotinstrument"
	ordergrpc "github.com/exchange-grpc/internal/adapter/grpc/server/order"
	spotgrpc "github.com/exchange-grpc/internal/adapter/grpc/server/spotinstrument"
	"github.com/exchange-grpc/internal/adapter/repository/memory"
	"github.com/exchange-grpc/internal/domain"
	"github.com/exchange-grpc/internal/platform/interceptor"
	orderuc "github.com/exchange-grpc/internal/usecase/order"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func TestServer_CreateOrder_success(t *testing.T) {
	client, cleanup := setupOrderClient(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := client.CreateOrder(ctx, &exchangev1.CreateOrderRequest{
		UserId:    "user-1",
		MarketId:  "BTC-USDT",
		OrderType: exchangev1.OrderType_ORDER_TYPE_LIMIT,
		Price:     "42000",
		Quantity:  "0.01",
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	if resp.GetOrderId() == "" {
		t.Fatal("expected order id")
	}
	if resp.GetStatus() != "created" {
		t.Fatalf("Status = %q, want created", resp.GetStatus())
	}

	statusResp, err := client.GetOrderStatus(ctx, &exchangev1.GetOrderStatusRequest{
		OrderId: resp.GetOrderId(),
		UserId:  "user-1",
	})
	if err != nil {
		t.Fatalf("GetOrderStatus() error = %v", err)
	}
	if statusResp.GetMarketId() != "BTC-USDT" {
		t.Fatalf("MarketId = %q", statusResp.GetMarketId())
	}
}

func TestServer_CreateOrder_inactiveMarket(t *testing.T) {
	client, cleanup := setupOrderClient(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := client.CreateOrder(ctx, &exchangev1.CreateOrderRequest{
		UserId:    "user-1",
		MarketId:  "SOL-USDT",
		OrderType: exchangev1.OrderType_ORDER_TYPE_MARKET,
		Quantity:  "1",
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status = %v, want FailedPrecondition", status.Code(err))
	}
}

func TestServer_CreateOrder_forbiddenMarket(t *testing.T) {
	client, cleanup := setupOrderClient(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := client.CreateOrder(ctx, &exchangev1.CreateOrderRequest{
		UserId:    "user-1",
		MarketId:  "BNB-USDT",
		OrderType: exchangev1.OrderType_ORDER_TYPE_MARKET,
		Quantity:  "1",
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status = %v, want PermissionDenied", status.Code(err))
	}

	resp, err := client.CreateOrder(ctx, &exchangev1.CreateOrderRequest{
		UserId:    "user-1",
		MarketId:  "BNB-USDT",
		OrderType: exchangev1.OrderType_ORDER_TYPE_MARKET,
		Quantity:  "1",
		UserRoles: []string{"trader"},
	})
	if err != nil {
		t.Fatalf("CreateOrder(trader) error = %v", err)
	}
	if resp.GetOrderId() == "" {
		t.Fatal("expected order id")
	}
}

func TestServer_CreateOrder_deletedMarket(t *testing.T) {
	client, cleanup := setupOrderClient(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := client.CreateOrder(ctx, &exchangev1.CreateOrderRequest{
		UserId:    "user-1",
		MarketId:  "XRP-USDT",
		OrderType: exchangev1.OrderType_ORDER_TYPE_MARKET,
		Quantity:  "1",
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status = %v, want FailedPrecondition", status.Code(err))
	}
}

type orderTestEnv struct {
	Client   exchangev1.OrderServiceClient
	Services ordergrpc.Services
}

func TestServer_StreamOrderUpdates_receivesStatusChanges(t *testing.T) {
	env, cleanup := setupOrderEnv(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	created, err := env.Client.CreateOrder(ctx, &exchangev1.CreateOrderRequest{
		UserId:    "user-1",
		MarketId:  "BTC-USDT",
		OrderType: exchangev1.OrderType_ORDER_TYPE_MARKET,
		Quantity:  "1",
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	stream, err := env.Client.StreamOrderUpdates(ctx, &exchangev1.StreamOrderUpdatesRequest{
		OrderId: created.GetOrderId(),
		UserId:  "user-1",
	})
	if err != nil {
		t.Fatalf("StreamOrderUpdates() error = %v", err)
	}

	first, err := stream.Recv()
	if err != nil {
		t.Fatalf("first Recv() error = %v", err)
	}
	if first.GetStatus() != "created" {
		t.Fatalf("first status = %q, want created", first.GetStatus())
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = env.Services.UpdateOrderStatus.Execute(context.Background(), orderuc.UpdateOrderStatusInput{
			OrderID: created.GetOrderId(),
			UserID:  "user-1",
			Status:  domain.OrderStatusFilled,
		})
	}()

	second, err := stream.Recv()
	if err != nil {
		t.Fatalf("second Recv() error = %v", err)
	}
	if second.GetStatus() != "filled" {
		t.Fatalf("second status = %q, want filled", second.GetStatus())
	}
}

func TestServer_GetOrderStatus_wrongUser(t *testing.T) {
	client, cleanup := setupOrderClient(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	created, err := client.CreateOrder(ctx, &exchangev1.CreateOrderRequest{
		UserId:    "user-1",
		MarketId:  "ETH-USDT",
		OrderType: exchangev1.OrderType_ORDER_TYPE_MARKET,
		Quantity:  "1",
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	_, err = client.GetOrderStatus(ctx, &exchangev1.GetOrderStatusRequest{
		OrderId: created.GetOrderId(),
		UserId:  "user-2",
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("status = %v, want NotFound", status.Code(err))
	}
}

func setupOrderClient(t *testing.T) (exchangev1.OrderServiceClient, func()) {
	env, cleanup := setupOrderEnv(t)
	return env.Client, cleanup
}

func setupOrderEnv(t *testing.T) (orderTestEnv, func()) {
	t.Helper()

	spotListener := bufconn.Listen(bufSize)
	spotRepo := memory.NewSeededMarketRepository()
	spotServer := spotgrpc.NewServerFromRepository(spotRepo)

	log := zap.NewNop()
	interceptors := interceptor.ChainUnaryServer(
		interceptor.UnaryServerRequestID,
		interceptor.UnaryServerLogger(log),
		interceptor.UnaryServerPanicRecovery(log),
	)

	spotGRPC := grpc.NewServer(grpc.UnaryInterceptor(interceptors))
	exchangev1.RegisterSpotInstrumentServiceServer(spotGRPC, spotServer)

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
	orderGRPC := grpc.NewServer(grpc.UnaryInterceptor(interceptors))
	exchangev1.RegisterOrderServiceServer(orderGRPC, orderServer)

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

	cleanup := func() {
		orderConn.Close()
		spotConn.Close()
		orderGRPC.Stop()
		spotGRPC.Stop()
	}

	return orderTestEnv{
		Client:   exchangev1.NewOrderServiceClient(orderConn),
		Services: orderServices,
	}, cleanup
}
