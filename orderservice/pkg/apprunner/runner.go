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

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/postgres"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/ratelimit"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/spotclient"
	grpcserver "github.com/exchange-grpc/orderservice/internal/interfaces/grpcserver"
	"github.com/exchange-grpc/orderservice/pkg/config"
	orderv1 "github.com/exchange-grpc/proto/pb/order/v1"
	sharedapprunner "github.com/exchange-grpc/shared/apprunner"
	"github.com/exchange-grpc/shared/grpc"
	sharedhealth "github.com/exchange-grpc/shared/health"
	"github.com/exchange-grpc/shared/logger"
	sharedmetrics "github.com/exchange-grpc/shared/metrics"
	sharedredis "github.com/exchange-grpc/shared/redis"
	"github.com/exchange-grpc/shared/sessionvalidation"
	"github.com/exchange-grpc/shared/tracing"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
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
	log, level, err := logger.New()
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()
	admin := logger.ServeLevelAdmin(r.cfg.LogLevelAddr, level, log)
	defer logger.ShutdownLevelAdmin(admin, log)
	metricsSrv := sharedmetrics.Serve(r.cfg.MetricsAddr, log)
	defer sharedmetrics.Shutdown(metricsSrv, log)

	if err := r.run(log); err != nil {
		log.Fatal("orderservice failed", zap.Error(err))
	}
}

func (r *AppRunner) run(log *zap.Logger) error {
	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownTracing, err := tracing.Setup(runCtx, "orderservice")
	if err != nil {
		return fmt.Errorf("init tracing: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), tracing.DefaultShutdownTimeout)
		defer cancel()
		_ = shutdownTracing(shutdownCtx)
	}()

	migrationsDir := resolveMigrationsDir(r.cfg.MigrationsDir)

	db, err := postgres.Connect(runCtx, r.cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	if err := postgres.RunMigrations(runCtx, db, migrationsDir); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	tokens, err := sessionvalidation.NewTokenService(r.cfg.JWTSecret, r.cfg.AccessTokenTTL)
	if err != nil {
		return fmt.Errorf("init token service: %w", err)
	}
	validator, err := grpc.NewProtoValidator()
	if err != nil {
		return fmt.Errorf("init proto validator: %w", err)
	}

	spotConn, err := spotclient.Dial(runCtx, r.cfg.SpotServiceHost)
	if err != nil {
		return fmt.Errorf("dial spot service: %w", err)
	}
	defer spotConn.Close()

	marketClient := spotclient.New(spotConn, r.cfg.SpotGRPCTimeout)
	orderRepo := postgres.NewOrderRepository(db)
	idempotencyStore := postgres.NewIdempotencyStore(db, r.cfg.IdempotencyTTL)

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
	redisClient, err = sharedredis.Connect(
		runCtx,
		r.cfg.RedisURL,
		sharedredis.WithPoolSize(r.cfg.RedisPoolSize),
		sharedredis.WithMaxRetries(r.cfg.RedisMaxRetries),
	)
	if err != nil {
		// In-memory fallback is intentional for local/dev when Redis is down.
		// Limits are per-process: N replicas multiply the effective budget and
		// weaken brute-force protection. Production should keep Redis required.
		log.Warn("redis unavailable, using in-memory create order rate limiter", zap.Error(err))
		createOrderLimiter = ratelimit.NewCreateOrderLimiter(rateLimitCfg)
	} else {
		defer redisClient.Close()
		createOrderLimiter = ratelimit.NewRedisCreateOrderLimiter(redisClient.Raw(), rateLimitCfg)
		log.Info("create order rate limiter uses redis", zap.String("redis_url", r.cfg.RedisURL))
	}

	orderServices := grpcserver.NewServices(orderRepo, idempotencyStore, marketClient, createOrderLimiter, r.cfg.OrderHubBufferSize, r.cfg.OrderHubPublishTimeout, log)
	server := grpcserver.NewServer(orderServices)

	grpcServer := googlegrpc.NewServer(
		googlegrpc.StatsHandler(otelgrpc.NewServerHandler()),
		googlegrpc.UnaryInterceptor(grpc.UnaryServerInterceptors(
			log,
			validator,
			tokens,
			grpc_health_v1.Health_Check_FullMethodName,
		)),
		googlegrpc.StreamInterceptor(grpc.StreamServerInterceptors(
			log,
			validator,
			tokens,
			grpc_health_v1.Health_Watch_FullMethodName,
		)),
	)
	orderv1.RegisterOrderServiceServer(grpcServer, server)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	listener, err := net.Listen("tcp", r.cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", r.cfg.GRPCAddr, err)
	}

	criticalChecks := []sharedhealth.Checker{db.Ping}
	var optionalChecks []sharedhealth.Checker
	if redisClient != nil {
		optionalChecks = append(optionalChecks, redisClient.Ping)
	}
	healthWatcher := sharedhealth.NewWatcher(
		healthServer,
		orderv1.OrderService_ServiceDesc.ServiceName,
		10*time.Second,
		r.cfg.HealthCheckTimeout,
		log,
		criticalChecks,
		optionalChecks...,
	)
	if err := healthWatcher.Check(runCtx); err != nil {
		_ = listener.Close()
		return fmt.Errorf("initial health check: %w", err)
	}
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
