package apprunner

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	userv1 "github.com/exchange-grpc/proto/pb/user/v1"
	"github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/shared/logger"
	"github.com/exchange-grpc/shared/sessionvalidation"
	"github.com/exchange-grpc/userservice/internal/application"
	grpcserver "github.com/exchange-grpc/userservice/internal/interfaces/grpcserver"
	"github.com/exchange-grpc/userservice/internal/infrastructure/memory"
	"github.com/exchange-grpc/userservice/pkg/config"
	"go.uber.org/zap"
	googlegrpc "google.golang.org/grpc"
)

// AppRunner запускает и останавливает userservice.
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
		log.Fatal("userservice failed", zap.Error(err))
	}
}

func (r *AppRunner) run(log *zap.Logger) error {
	tokens, err := sessionvalidation.NewTokenService(r.cfg.JWTSecret, r.cfg.JWTTTL)
	if err != nil {
		return fmt.Errorf("init token service: %w", err)
	}

	userRepo := memory.NewUserRepository()
	registerUC := application.NewRegister(userRepo)
	loginUC := application.NewLogin(userRepo, tokens)
	server := grpcserver.NewServer(registerUC, loginUC)

	grpcServer := googlegrpc.NewServer(
		googlegrpc.UnaryInterceptor(grpc.ChainUnaryServer(
			grpc.UnaryServerRequestID,
			grpc.NewUnaryServerJWTAuth(tokens,
				userv1.UserService_Register_FullMethodName,
				userv1.UserService_Login_FullMethodName,
			),
		)),
	)
	userv1.RegisterUserServiceServer(grpcServer, server)

	listener, err := net.Listen("tcp", r.cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", r.cfg.GRPCAddr, err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Info("userservice started", zap.String("addr", r.cfg.GRPCAddr))
		errCh <- grpcServer.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down userservice")
		grpcServer.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}
