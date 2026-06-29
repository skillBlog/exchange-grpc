package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	spotgrpc "github.com/exchange-grpc/internal/adapter/grpc/server/spotinstrument"
	"github.com/exchange-grpc/internal/adapter/repository/memory"
	"github.com/exchange-grpc/internal/platform/cache"
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
		log.Fatal("spot instrument service failed", zap.Error(err))
	}
}

func run(cfg config.Config, log *zap.Logger) error {
	shutdownTracing, err := tracing.Init(context.Background(), "spot-instrument-service", cfg.TracingOTLPEndpoint)
	if err != nil {
		return fmt.Errorf("init tracing: %w", err)
	}
	defer func() { _ = shutdownTracing(context.Background()) }()
	if tracing.Enabled() {
		log.Info("tracing enabled", zap.String("otlp_endpoint", cfg.TracingOTLPEndpoint))
	}

	marketRepo := memory.NewSeededMarketRepository()
	server := spotgrpc.NewServerFromRepository(marketRepo)

	chain := []grpc.UnaryServerInterceptor{
		interceptor.UnaryServerRequestID,
		interceptor.UnaryServerSpanRequestID,
		interceptor.UnaryServerLogger(log),
		interceptor.UnaryServerPanicRecovery(log),
		metrics.UnaryServerInterceptor(),
	}

	if cfg.RedisAddr != "" {
		redisCache, err := cache.NewRedisResponseCache(cfg.RedisAddr)
		if err != nil {
			return fmt.Errorf("connect redis: %w", err)
		}
		defer func() { _ = redisCache.Close() }()

		log.Info("redis cache enabled",
			zap.String("addr", cfg.RedisAddr),
			zap.Duration("ttl", cfg.RedisCacheTTL),
		)
		chain = append(chain, interceptor.UnaryServerRedisCache(redisCache, cfg.RedisCacheTTL, log))
	}

	serverOpts := tracing.ServerOptions()
	serverOpts = append(serverOpts, grpc.UnaryInterceptor(interceptor.ChainUnaryServer(chain...)))
	grpcServer := grpc.NewServer(serverOpts...)
	exchangev1.RegisterSpotInstrumentServiceServer(grpcServer, server)
	metrics.RegisterGRPC(grpcServer)

	listener, err := net.Listen("tcp", cfg.SpotInstrumentAddr)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 2)
	go func() {
		log.Info("metrics server started", zap.String("addr", cfg.SpotMetricsAddr))
		errCh <- metrics.StartHTTPServer(ctx, cfg.SpotMetricsAddr)
	}()
	go func() {
		log.Info("spot instrument service started", zap.String("addr", cfg.SpotInstrumentAddr))
		errCh <- grpcServer.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down spot instrument service")
		grpcServer.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}
