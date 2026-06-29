package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	spotclient "github.com/exchange-grpc/internal/adapter/client/spotinstrument"
	ordergrpc "github.com/exchange-grpc/internal/adapter/grpc/server/order"
	"github.com/exchange-grpc/internal/adapter/repository/memory"
	"github.com/exchange-grpc/internal/platform/config"
	"github.com/exchange-grpc/internal/platform/interceptor"
	"github.com/exchange-grpc/internal/platform/logger"
	"github.com/exchange-grpc/internal/platform/metrics"
	"github.com/exchange-grpc/internal/platform/tracing"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load()

	log, err := logger.New()
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()

	if err := run(cfg, log); err != nil {
		log.Fatal("order service failed", zap.Error(err))
	}
}

func run(cfg config.Config, log *zap.Logger) error {
	shutdownTracing, err := tracing.Init(context.Background(), "order-service", cfg.TracingOTLPEndpoint)
	if err != nil {
		return fmt.Errorf("init tracing: %w", err)
	}
	defer func() { _ = shutdownTracing(context.Background()) }()
	if tracing.Enabled() {
		log.Info("tracing enabled", zap.String("otlp_endpoint", cfg.TracingOTLPEndpoint))
	}

	spotConn, err := spotclient.Dial(context.Background(), cfg.SpotInstrumentHost)
	if err != nil {
		return fmt.Errorf("dial spot instrument service: %w", err)
	}
	defer spotConn.Close()

	marketClient := spotclient.New(spotConn)
	orderRepo := memory.NewOrderRepository()
	orderServices := ordergrpc.NewServices(orderRepo, marketClient)
	server := ordergrpc.NewServer(orderServices)

	serverOpts := tracing.ServerOptions()
	serverOpts = append(serverOpts, grpc.UnaryInterceptor(interceptor.ChainUnaryServer(
		interceptor.UnaryServerRequestID,
		interceptor.UnaryServerSpanRequestID,
		interceptor.UnaryServerLogger(log),
		interceptor.UnaryServerPanicRecovery(log),
		metrics.UnaryServerInterceptor(),
	)))
	grpcServer := grpc.NewServer(serverOpts...)
	exchangev1.RegisterOrderServiceServer(grpcServer, server)
	metrics.RegisterGRPC(grpcServer)

	listener, err := net.Listen("tcp", cfg.OrderAddr)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 2)
	go func() {
		log.Info("metrics server started", zap.String("addr", cfg.OrderMetricsAddr))
		errCh <- metrics.StartHTTPServer(ctx, cfg.OrderMetricsAddr)
	}()
	go func() {
		log.Info("order service started",
			zap.String("addr", cfg.OrderAddr),
			zap.String("spot_instrument", cfg.SpotInstrumentHost),
		)
		errCh <- grpcServer.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down order service")
		grpcServer.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}
