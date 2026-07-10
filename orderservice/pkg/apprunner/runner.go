package apprunner

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	orderv1 "github.com/exchange-grpc/proto/pb/order/v1"
	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/postgres"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/ratelimit"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/spotclient"
	grpcserver "github.com/exchange-grpc/orderservice/internal/interfaces/grpcserver"
	"github.com/exchange-grpc/orderservice/pkg/config"
	sharedapprunner "github.com/exchange-grpc/shared/apprunner"
	sharedhealth "github.com/exchange-grpc/shared/health"
	"github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/shared/logger"
	sharedredis "github.com/exchange-grpc/shared/redis"
	"github.com/exchange-grpc/shared/sessionvalidation"
	"go.uber.org/zap"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
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
	ctx := context.Background()
	migrationsDir := resolveMigrationsDir(r.cfg.MigrationsDir)

	if err := postgres.RunMigrations(ctx, r.cfg.DatabaseURL, migrationsDir); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	db, err := postgres.Connect(ctx, r.cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	tokens, err := sessionvalidation.NewTokenService(r.cfg.JWTSecret, r.cfg.AccessTokenTTL)
	if err != nil {
		return fmt.Errorf("init token service: %w", err)
	}
	validator := grpc.MustNewProtoValidator()

	spotConn, err := spotclient.Dial(context.Background(), r.cfg.SpotServiceHost)
	if err != nil {
		return fmt.Errorf("dial spot service: %w", err)
	}
	defer spotConn.Close()

	marketClient := spotclient.New(spotConn, r.cfg.SpotGRPCTimeout)
	orderRepo := postgres.NewOrderRepository(db)
	idempotencyStore := postgres.NewIdempotencyStore(db)

	rateLimitCfg := application.CreateOrderRateLimitConfig{
		GlobalLimit:  r.cfg.CreateOrderRateLimit.GlobalLimit,
		GlobalWindow: r.cfg.CreateOrderRateLimit.GlobalWindow,
		BasicLimit:   r.cfg.CreateOrderRateLimit.BasicLimit,
		PremiumLimit: r.cfg.CreateOrderRateLimit.PremiumLimit,
		AdminLimit:   r.cfg.CreateOrderRateLimit.AdminLimit,
		UserWindow:   r.cfg.CreateOrderRateLimit.UserWindow,
	}

	var createOrderLimiter application.CreateOrderRateLimiter
	var redisClient *sharedredis.Client
	redisClient, err = sharedredis.Connect(ctx, r.cfg.RedisURL)
	if err != nil {
		log.Warn("redis unavailable, using in-memory create order rate limiter", zap.Error(err))
		createOrderLimiter = ratelimit.NewCreateOrderLimiter(rateLimitCfg)
	} else {
		defer redisClient.Close()
		createOrderLimiter = ratelimit.NewRedisCreateOrderLimiter(redisClient.Raw(), rateLimitCfg)
		log.Info("create order rate limiter uses redis", zap.String("redis_url", r.cfg.RedisURL))
	}

	orderServices := grpcserver.NewServices(orderRepo, idempotencyStore, marketClient, createOrderLimiter, r.cfg.OrderHubBufferSize, log)
	server := grpcserver.NewServer(orderServices)

	grpcServer := googlegrpc.NewServer(
		googlegrpc.UnaryInterceptor(grpc.ChainUnaryServer(
			grpc.UnaryServerRequestID,
			grpc.UnaryServerLogging(log),
			grpc.NewUnaryServerProtoValidate(validator),
			grpc.NewUnaryServerJWTAuth(tokens),
		)),
		googlegrpc.ChainStreamInterceptor(
			grpc.StreamServerRequestID,
			grpc.StreamServerLogging(log),
			grpc.NewStreamServerJWTAuth(tokens),
		),
	)
	orderv1.RegisterOrderServiceServer(grpcServer, server)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(orderv1.OrderService_ServiceDesc.ServiceName, grpc_health_v1.HealthCheckResponse_SERVING)

	listener, err := net.Listen("tcp", r.cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", r.cfg.GRPCAddr, err)
	}

	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	healthChecks := []sharedhealth.Checker{db.Ping}
	if redisClient != nil {
		healthChecks = append(healthChecks, redisClient.Ping)
	}
	healthWatcher := sharedhealth.NewWatcher(
		healthServer,
		orderv1.OrderService_ServiceDesc.ServiceName,
		10*time.Second,
		log,
		healthChecks...,
	)
	go healthWatcher.Run(runCtx)

	return sharedapprunner.ServeUntilSignal(
		runCtx,
		grpcServer,
		listener,
		log,
		"orderservice",
		sharedapprunner.DefaultShutdownTimeout,
	)
}

func resolveMigrationsDir(dir string) string {
	if filepath.IsAbs(dir) {
		return dir
	}
	candidates := []string{dir, filepath.Join("orderservice", dir)}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return dir
}
