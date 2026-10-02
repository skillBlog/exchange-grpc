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

	userv1 "github.com/exchange-grpc/proto/pb/user/v1"
	sharedapprunner "github.com/exchange-grpc/shared/apprunner"
	"github.com/exchange-grpc/shared/grpc"
	sharedhealth "github.com/exchange-grpc/shared/health"
	"github.com/exchange-grpc/shared/logger"
	sharedmetrics "github.com/exchange-grpc/shared/metrics"
	sharedredis "github.com/exchange-grpc/shared/redis"
	"github.com/exchange-grpc/shared/sessionvalidation"
	"github.com/exchange-grpc/shared/tracing"
	"github.com/exchange-grpc/userservice/internal/application"
	"github.com/exchange-grpc/userservice/internal/infrastructure/bcrypt"
	"github.com/exchange-grpc/userservice/internal/infrastructure/postgres"
	"github.com/exchange-grpc/userservice/internal/infrastructure/ratelimit"
	grpcserver "github.com/exchange-grpc/userservice/internal/interfaces/grpcserver"
	"github.com/exchange-grpc/userservice/pkg/config"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
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
		log.Fatal("userservice failed", zap.Error(err))
	}
}

func (r *AppRunner) run(log *zap.Logger) error {
	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownTracing, err := tracing.Setup(runCtx, "userservice")
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

	accessTokens, err := sessionvalidation.NewTokenService(r.cfg.JWTSecret, r.cfg.AccessTokenTTL)
	if err != nil {
		return fmt.Errorf("init access token service: %w", err)
	}
	validator, err := grpc.NewProtoValidator()
	if err != nil {
		return fmt.Errorf("init proto validator: %w", err)
	}

	userRepo := postgres.NewUserRepository(db)
	refreshRepo := postgres.NewRefreshTokenRepository(db)
	hasher := bcrypt.NewHasher(r.cfg.BcryptCost)
	refreshTokens := sessionvalidation.NewRefreshTokenService(refreshRepo, r.cfg.RefreshTokenTTL)

	var loginLimiter application.LoginRateLimiter
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
		log.Error("redis unavailable, using in-memory login rate limiter", zap.Error(err))
		loginLimiter = ratelimit.NewLoginLimiter(r.cfg.LoginRateLimit, r.cfg.LoginRateWindow)
	} else {
		defer redisClient.Close()
		loginLimiter = ratelimit.NewRedisLoginLimiter(redisClient.Raw(), r.cfg.LoginRateLimit, r.cfg.LoginRateWindow)
		log.Info("login rate limiter uses redis", zap.String("redis_url", r.cfg.RedisURL))
	}

	registerUC := application.NewRegister(userRepo, hasher, accessTokens, refreshTokens, log)
	loginUC := application.NewLogin(userRepo, hasher, accessTokens, refreshTokens, loginLimiter, log)
	refreshUC := application.NewRefreshToken(userRepo, accessTokens, refreshTokens, log)
	getUserUC := application.NewGetUser(userRepo)
	logoutUC := application.NewLogout(refreshTokens, log)
	server := grpcserver.NewServer(registerUC, loginUC, refreshUC, getUserUC, logoutUC)

	grpcServer := googlegrpc.NewServer(
		googlegrpc.StatsHandler(otelgrpc.NewServerHandler()),
		googlegrpc.UnaryInterceptor(grpc.UnaryServerInterceptors(
			log,
			validator,
			accessTokens,
			userv1.UserService_Register_FullMethodName,
			userv1.UserService_Login_FullMethodName,
			userv1.UserService_RefreshToken_FullMethodName,
			userv1.UserService_Logout_FullMethodName,
			grpc_health_v1.Health_Check_FullMethodName,
		)),
		googlegrpc.StreamInterceptor(grpc.StreamServerInterceptors(
			log,
			validator,
			accessTokens,
			grpc_health_v1.Health_Watch_FullMethodName,
		)),
	)
	userv1.RegisterUserServiceServer(grpcServer, server)

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
		userv1.UserService_ServiceDesc.ServiceName,
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
		"userservice",
		sharedapprunner.DefaultShutdownTimeout,
	)
}

func resolveMigrationsDir(dir string) string {
	if filepath.IsAbs(dir) {
		return dir
	}
	candidates := []string{dir, filepath.Join("userservice", dir)}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return dir
}
