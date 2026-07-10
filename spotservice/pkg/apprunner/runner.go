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
	sharedhealth "github.com/exchange-grpc/shared/health"
	"github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/shared/logger"
	"github.com/exchange-grpc/shared/sessionvalidation"
	grpcserver "github.com/exchange-grpc/spotservice/internal/interfaces/grpcserver"
	"github.com/exchange-grpc/spotservice/internal/infrastructure/cache"
	"github.com/exchange-grpc/spotservice/internal/infrastructure/postgres"
	"github.com/exchange-grpc/spotservice/internal/infrastructure/ratelimit"
	"github.com/exchange-grpc/spotservice/pkg/config"
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

	marketRepo := cache.NewMarketRepository(postgres.NewMarketRepository(db), r.cfg.MarketCacheTTL)
	viewMarketsLimiter := ratelimit.NewViewMarketsLimiter(r.cfg.ViewMarketsRateLimit, r.cfg.ViewMarketsRateWindow)
	server := grpcserver.NewServerFromRepository(marketRepo, viewMarketsLimiter)

	grpcServer := googlegrpc.NewServer(
		googlegrpc.UnaryInterceptor(grpc.ChainUnaryServer(
			grpc.UnaryServerRequestID,
			grpc.UnaryServerLogging(log),
			grpc.NewUnaryServerProtoValidate(validator),
			grpc.NewUnaryServerJWTAuth(tokens),
		)),
	)
	spotv1.RegisterSpotServiceServer(grpcServer, server)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(spotv1.SpotService_ServiceDesc.ServiceName, grpc_health_v1.HealthCheckResponse_SERVING)

	listener, err := net.Listen("tcp", r.cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", r.cfg.GRPCAddr, err)
	}

	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	healthWatcher := sharedhealth.NewWatcher(
		healthServer,
		spotv1.SpotService_ServiceDesc.ServiceName,
		10*time.Second,
		log,
		db.Ping,
		marketRepo.Ping,
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
