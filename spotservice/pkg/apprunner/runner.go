package apprunner

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	spotv1 "github.com/exchange-grpc/proto/pb/spot/v1"
	"github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/shared/logger"
	"github.com/exchange-grpc/shared/sessionvalidation"
	grpcserver "github.com/exchange-grpc/spotservice/internal/interfaces/grpcserver"
	"github.com/exchange-grpc/spotservice/internal/infrastructure/memory"
	"github.com/exchange-grpc/spotservice/pkg/config"
	"go.uber.org/zap"
	googlegrpc "google.golang.org/grpc"
)

// AppRunner запускает и останавливает spotservice.
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
		log.Fatal("spotservice failed", zap.Error(err))
	}
}

func (r *AppRunner) run(log *zap.Logger) error {
	tokens, err := sessionvalidation.NewTokenService(r.cfg.JWTSecret, r.cfg.JWTTTL)
	if err != nil {
		return fmt.Errorf("init token service: %w", err)
	}

	marketRepo := memory.NewSeededMarketRepository()
	server := grpcserver.NewServerFromRepository(marketRepo)

	grpcServer := googlegrpc.NewServer(
		googlegrpc.UnaryInterceptor(grpc.ChainUnaryServer(
			grpc.UnaryServerRequestID,
			grpc.NewUnaryServerJWTAuth(tokens),
		)),
	)
	spotv1.RegisterSpotServiceServer(grpcServer, server)

	listener, err := net.Listen("tcp", r.cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", r.cfg.GRPCAddr, err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Info("spotservice started", zap.String("addr", r.cfg.GRPCAddr))
		errCh <- grpcServer.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down spotservice")
		grpcServer.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}
