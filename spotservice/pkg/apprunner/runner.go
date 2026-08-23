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

	spotv1 "github.com/exchange-grpc/proto/pb/spot/v1"
	sharedapprunner "github.com/exchange-grpc/shared/apprunner"
	"github.com/exchange-grpc/shared/grpc"
	sharedhealth "github.com/exchange-grpc/shared/health"
	"github.com/exchange-grpc/shared/logger"
	sharedredis "github.com/exchange-grpc/shared/redis"
	"github.com/exchange-grpc/shared/sessionvalidation"
	"github.com/exchange-grpc/shared/tracing"
	"github.com/exchange-grpc/spotservice/internal/application"
	"github.com/exchange-grpc/spotservice/internal/infrastructure/cache"
	"github.com/exchange-grpc/spotservice/internal/infrastructure/postgres"
	"github.com/exchange-grpc/spotservice/internal/infrastructure/ratelimit"
	grpcserver "github.com/exchange-grpc/spotservice/internal/interfaces/grpcserver"
	"github.com/exchange-grpc/spotservice/pkg/config"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
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
	log, level, err := logger.New()
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()
	logger.ServeLevelAdmin(r.cfg.LogLevelAddr, level, log)

	if err := r.run(log); err != nil {
		log.Fatal("spotservice failed", zap.Error(err))
	}
}

func (r *AppRunner) run(log *zap.Logger) error {
	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownTracing, err := tracing.Setup(runCtx, "spotservice")
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

	marketRepo := cache.NewMarketRepository(postgres.NewMarketRepository(db), r.cfg.MarketCacheTTL)

	var viewMarketsLimiter application.ViewMarketsRateLimiter
	var redisClient *sharedredis.Client
	redisClient, err = sharedredis.Connect(
		runCtx,
		r.cfg.RedisURL,
		sharedredis.WithPoolSize(r.cfg.RedisPoolSize),
		sharedredis.WithMaxRetries(r.cfg.RedisMaxRetries),
	)
	if err != nil {
		log.Warn("redis unavailable, using in-memory view markets rate limiter", zap.Error(err))
		viewMarketsLimiter = ratelimit.NewViewMarketsLimiter(r.cfg.ViewMarketsRateLimit, r.cfg.ViewMarketsRateWindow)
	} else {
		defer redisClient.Close()
		viewMarketsLimiter = ratelimit.NewRedisViewMarketsLimiter(redisClient.Raw(), r.cfg.ViewMarketsRateLimit, r.cfg.ViewMarketsRateWindow)
		log.Info("view markets rate limiter uses redis", zap.String("redis_url", r.cfg.RedisURL))
	}

	server := grpcserver.NewServerFromRepository(marketRepo, viewMarketsLimiter, log)

	grpcServer := googlegrpc.NewServer(
		googlegrpc.StatsHandler(otelgrpc.NewServerHandler()),
		googlegrpc.UnaryInterceptor(grpc.UnaryServerInterceptors(
			log,
			validator,
			tokens,
			grpc_health_v1.Health_Check_FullMethodName,
			grpc_health_v1.Health_Watch_FullMethodName,
		)),
	)
	spotv1.RegisterSpotServiceServer(grpcServer, server)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus(spotv1.SpotService_ServiceDesc.ServiceName, grpc_health_v1.HealthCheckResponse_SERVING)

	listener, err := net.Listen("tcp", r.cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", r.cfg.GRPCAddr, err)
	}

	healthWatcher := sharedhealth.NewWatcher(
		healthServer,
		spotv1.SpotService_ServiceDesc.ServiceName,
		10*time.Second,
		r.cfg.HealthCheckTimeout,
		log,
		[]sharedhealth.Checker{db.Ping},
	)
	go healthWatcher.Run(runCtx)

	return sharedapprunner.ServeUntilSignal(
		runCtx,
		grpcServer,
		listener,
		log,
		"spotservice",
		sharedapprunner.DefaultShutdownTimeout,
	)
}

func resolveMigrationsDir(dir string) string {
	if filepath.IsAbs(dir) {
		return dir
	}
	candidates := []string{dir, filepath.Join("spotservice", dir)}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return dir
}
