package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	orderclient "github.com/exchange-grpc/internal/adapter/client/order"
	"github.com/exchange-grpc/internal/platform/config"
	"github.com/exchange-grpc/internal/platform/interceptor"
	"github.com/exchange-grpc/internal/platform/tracing"
	"github.com/google/uuid"
)

func main() {
	cfg := config.Load()

	userID := flag.String("user-id", "", "user identifier")
	marketID := flag.String("market-id", "", "market identifier (e.g. BTC-USDT)")
	orderType := flag.String("order-type", "limit", "order type: limit or market")
	price := flag.String("price", "", "order price (required for limit orders)")
	quantity := flag.String("quantity", "", "order quantity")
	addr := flag.String("addr", cfg.OrderHost, "OrderService gRPC address")
	requestID := flag.String("request-id", "", "x-request-id (generated when empty)")
	userRoles := flag.String("user-roles", "", "comma-separated user roles (e.g. trader,admin)")
	flag.Parse()

	if err := run(cfg, *userID, *marketID, *orderType, *price, *quantity, *addr, *requestID, parseRoles(*userRoles)); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, userID, marketID, orderType, price, quantity, addr, requestID string, userRoles []string) error {
	shutdownTracing, err := tracing.Init(context.Background(), "order-client", cfg.TracingOTLPEndpoint)
	if err != nil {
		return fmt.Errorf("init tracing: %w", err)
	}
	defer func() { _ = shutdownTracing(context.Background()) }()

	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("user-id is required")
	}
	if strings.TrimSpace(marketID) == "" {
		return fmt.Errorf("market-id is required")
	}
	if strings.TrimSpace(quantity) == "" {
		return fmt.Errorf("quantity is required")
	}

	protoOrderType, err := parseOrderType(orderType)
	if err != nil {
		return err
	}
	if protoOrderType == exchangev1.OrderType_ORDER_TYPE_LIMIT && strings.TrimSpace(price) == "" {
		return fmt.Errorf("price is required for limit orders")
	}

	if requestID == "" {
		requestID = uuid.NewString()
	}

	ctx, cancel := context.WithTimeout(
		interceptor.ContextWithRequestID(context.Background(), requestID),
		10*time.Second,
	)
	defer cancel()

	conn, err := orderclient.Dial(ctx, addr)
	if err != nil {
		return fmt.Errorf("dial order service: %w", err)
	}
	defer conn.Close()

	client := exchangev1.NewOrderServiceClient(conn)
	resp, err := client.CreateOrder(ctx, &exchangev1.CreateOrderRequest{
		UserId:    userID,
		MarketId:  marketID,
		OrderType: protoOrderType,
		Price:     price,
		Quantity:  quantity,
		UserRoles: userRoles,
	})
	if err != nil {
		return fmt.Errorf("create order: %w", err)
	}

	fmt.Printf("order_id: %s\n", resp.GetOrderId())
	fmt.Printf("status: %s\n", resp.GetStatus())
	fmt.Printf("request_id: %s\n", requestID)
	return nil
}

func parseOrderType(value string) (exchangev1.OrderType, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "limit":
		return exchangev1.OrderType_ORDER_TYPE_LIMIT, nil
	case "market":
		return exchangev1.OrderType_ORDER_TYPE_MARKET, nil
	default:
		return exchangev1.OrderType_ORDER_TYPE_UNSPECIFIED, fmt.Errorf("unsupported order-type %q (use limit or market)", value)
	}
}

func parseRoles(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	roles := make([]string, 0, len(parts))
	for _, part := range parts {
		if role := strings.TrimSpace(part); role != "" {
			roles = append(roles, role)
		}
	}
	return roles
}
