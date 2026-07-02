package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
	orderv1 "github.com/exchange-grpc/proto/pb/order/v1"
	userv1 "github.com/exchange-grpc/proto/pb/user/v1"
	"github.com/exchange-grpc/orderserviceclient/pkg/config"
	"github.com/exchange-grpc/orderserviceclient/pkg/envloader"
	"github.com/exchange-grpc/orderserviceclient/pkg/mapper"
	"github.com/exchange-grpc/shared/grpc"
	"github.com/google/uuid"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	envloader.LoadEnv()
	cfg := config.LoadConfig()

	email := flag.String("email", "", "user email")
	password := flag.String("password", "", "user password")
	register := flag.Bool("register", false, "register user before login")
	marketID := flag.String("market-id", "", "market identifier (e.g. BTC-USDT)")
	orderSide := flag.String("order-side", "buy", "order side: buy or sell")
	price := flag.String("price", "", "optional order price")
	quantity := flag.String("quantity", "", "order quantity")
	orderAddr := flag.String("addr", cfg.OrderServiceHost, "OrderService gRPC address")
	userAddr := flag.String("user-addr", cfg.UserServiceHost, "UserService gRPC address")
	requestID := flag.String("request-id", "", "x-request-id (generated when empty)")
	flag.Parse()

	if err := run(*email, *password, *register, *marketID, *orderSide, *price, *quantity, *orderAddr, *userAddr, *requestID); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(email, password string, register bool, marketID, orderSide, price, quantity, orderAddr, userAddr, requestID string) error {
	if strings.TrimSpace(email) == "" {
		return fmt.Errorf("email is required")
	}
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("password is required")
	}
	if strings.TrimSpace(marketID) == "" {
		return fmt.Errorf("market-id is required")
	}
	if strings.TrimSpace(quantity) == "" {
		return fmt.Errorf("quantity is required")
	}

	protoSide, err := parseOrderSide(orderSide)
	if err != nil {
		return err
	}

	if requestID == "" {
		requestID = uuid.NewString()
	}

	ctx, cancel := context.WithTimeout(
		grpc.ContextWithRequestID(context.Background(), requestID),
		15*time.Second,
	)
	defer cancel()

	userConn, err := googlegrpc.NewClient(userAddr,
		googlegrpc.WithTransportCredentials(insecure.NewCredentials()),
		googlegrpc.WithUnaryInterceptor(grpc.UnaryClientRequestID),
	)
	if err != nil {
		return fmt.Errorf("dial user service: %w", err)
	}
	defer userConn.Close()

	userClient := userv1.NewUserServiceClient(userConn)
	if register {
		if _, err := userClient.Register(ctx, &userv1.RegisterRequest{
			Email:    email,
			Password: password,
		}); err != nil {
			return fmt.Errorf("register: %w", err)
		}
		fmt.Println("user registered")
	}

	loginResp, err := userClient.Login(ctx, &userv1.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	ctx = grpc.OutgoingContextWithBearer(ctx, loginResp.GetAccessToken())

	orderConn, err := googlegrpc.NewClient(orderAddr,
		googlegrpc.WithTransportCredentials(insecure.NewCredentials()),
		googlegrpc.WithUnaryInterceptor(grpc.ChainUnaryClient(
			grpc.UnaryClientRequestID,
			grpc.UnaryClientForwardAuthorization,
		)),
	)
	if err != nil {
		return fmt.Errorf("dial order service: %w", err)
	}
	defer orderConn.Close()

	client := orderv1.NewOrderServiceClient(orderConn)
	resp, err := client.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		MarketId: marketID,
		Side:     protoSide,
		Price:    mapper.MoneyFromString(price),
		Quantity: mapper.DecimalFromString(quantity),
	})
	if err != nil {
		return fmt.Errorf("create order: %w", err)
	}

	fmt.Printf("order_id: %s\n", mapper.UuidToString(resp.GetOrderId()))
	fmt.Printf("status: %s\n", resp.GetStatus().String())
	fmt.Printf("request_id: %s\n", requestID)
	return nil
}

func parseOrderSide(value string) (commonv1.OrderSide, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "buy":
		return commonv1.OrderSide_ORDER_SIDE_BUY, nil
	case "sell":
		return commonv1.OrderSide_ORDER_SIDE_SELL, nil
	default:
		return commonv1.OrderSide_ORDER_SIDE_UNSPECIFIED, fmt.Errorf("unsupported order-side %q (use buy or sell)", value)
	}
}
