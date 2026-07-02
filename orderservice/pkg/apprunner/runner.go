package apprunner

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	orderv1 "github.com/exchange-grpc/proto/pb/order/v1"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/memory"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/spotclient"
	grpcserver "github.com/exchange-grpc/orderservice/internal/interfaces/grpcserver"
	"github.com/exchange-grpc/orderservice/pkg/config"
	"github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/shared/logger"
	"github.com/exchange-grpc/shared/sessionvalidation"
	"go.uber.org/zap"
	googlegrpc "google.golang.org/grpc"
)

// AppRunner запускает и останавливает orderservice.
type AppRunner struct {
	cfg config.Config
}

// NewAppRunner создаёт runner с заданной конфигурацией.
func NewAppRunner(cfg config.Config) *AppRunner {
	return &AppRunner{cfg: cfg}
}

// Run стартует gRPC-сервер и блокируется до сигнала завершения.
func (r *AppRunner) Run() {
	log, err := logger.New()
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()

	if err := r.run(log); err != nil {
		log.Fatal("orderservice failed", zap.Error(err))
	}
}

func (r *AppRunner) run(log *zap.Logger) error {
	tokens, err := sessionvalidation.NewTokenService(r.cfg.JWTSecret, r.cfg.JWTTTL)
	if err != nil {
		return fmt.Errorf("init token service: %w", err)
	}

	spotConn, err := spotclient.Dial(context.Background(), r.cfg.SpotServiceHost)
	if err != nil {
		return fmt.Errorf("dial spot service: %w", err)
	}
	defer spotConn.Close()

	marketClient := spotclient.New(spotConn, r.cfg.SpotGRPCTimeout)
	orderRepo := memory.NewOrderRepository()
	orderServices := grpcserver.NewServices(orderRepo, marketClient, r.cfg.OrderHubBufferSize)
	server := grpcserver.NewServer(orderServices)

	grpcServer := googlegrpc.NewServer(
		googlegrpc.UnaryInterceptor(grpc.ChainUnaryServer(
			grpc.UnaryServerRequestID,
			grpc.NewUnaryServerJWTAuth(tokens),
		)),
		googlegrpc.ChainStreamInterceptor(
			grpc.StreamServerRequestID,
			grpc.NewStreamServerJWTAuth(tokens),
		),
	)
	orderv1.RegisterOrderServiceServer(grpcServer, server)

	listener, err := net.Listen("tcp", r.cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", r.cfg.GRPCAddr, err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Info("orderservice started",
			zap.String("addr", r.cfg.GRPCAddr),
			zap.String("spot_service", r.cfg.SpotServiceHost),
		)
		errCh <- grpcServer.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down orderservice")
		grpcServer.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}
