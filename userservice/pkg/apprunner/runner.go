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
	sharedhealth "github.com/exchange-grpc/shared/health"
	"github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/shared/logger"
	"github.com/exchange-grpc/shared/sessionvalidation"
	"github.com/exchange-grpc/userservice/internal/application"
	grpcserver "github.com/exchange-grpc/userservice/internal/interfaces/grpcserver"
	"github.com/exchange-grpc/userservice/internal/infrastructure/bcrypt"
	"github.com/exchange-grpc/userservice/internal/infrastructure/postgres"
	"github.com/exchange-grpc/userservice/internal/infrastructure/ratelimit"
	"github.com/exchange-grpc/userservice/internal/infrastructure/tokens"
	"github.com/exchange-grpc/userservice/pkg/config"
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

	accessTokens, err := sessionvalidation.NewTokenService(r.cfg.JWTSecret, r.cfg.AccessTokenTTL)
	if err != nil {
		return fmt.Errorf("init access token service: %w", err)
	}
	validator := grpc.MustNewProtoValidator()

	userRepo := postgres.NewUserRepository(db)
	refreshRepo := postgres.NewRefreshTokenRepository(db)
	hasher := bcrypt.NewHasher()
	refreshTokens := tokens.NewRefreshTokenService(refreshRepo, r.cfg.RefreshTokenTTL)
	loginLimiter := ratelimit.NewLoginLimiter(r.cfg.LoginRateLimit, r.cfg.LoginRateWindow)

	registerUC := application.NewRegister(userRepo, hasher)
	loginUC := application.NewLogin(userRepo, hasher, accessTokens, refreshTokens, loginLimiter)
	refreshUC := application.NewRefreshToken(userRepo, accessTokens, refreshTokens)
	getUserUC := application.NewGetUser(userRepo)
	logoutUC := application.NewLogout(refreshTokens)
	server := grpcserver.NewServer(registerUC, loginUC, refreshUC, getUserUC, logoutUC)

	grpcServer := googlegrpc.NewServer(
		googlegrpc.UnaryInterceptor(grpc.ChainUnaryServer(
			grpc.UnaryServerRequestID,
			grpc.UnaryServerLogging(log),
			grpc.NewUnaryServerProtoValidate(validator),
			grpc.NewUnaryServerJWTAuth(accessTokens,
				userv1.UserService_Register_FullMethodName,
				userv1.UserService_Login_FullMethodName,
				userv1.UserService_RefreshToken_FullMethodName,
				userv1.UserService_Logout_FullMethodName,
			),
		)),
	)
	userv1.RegisterUserServiceServer(grpcServer, server)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(userv1.UserService_ServiceDesc.ServiceName, grpc_health_v1.HealthCheckResponse_SERVING)

	listener, err := net.Listen("tcp", r.cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", r.cfg.GRPCAddr, err)
	}

	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	healthWatcher := sharedhealth.NewWatcher(
		healthServer,
		userv1.UserService_ServiceDesc.ServiceName,
		10*time.Second,
		log,
		db.Ping,
	)
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
