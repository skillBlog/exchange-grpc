package apprunner

import (
	"context"
	"net"
	"time"

	"go.uber.org/zap"
	googlegrpc "google.golang.org/grpc"
)

const DefaultShutdownTimeout = 15 * time.Second

// ServeUntilSignal запускает gRPC-сервер и выполняет graceful shutdown при отмене ctx.
func ServeUntilSignal(
	ctx context.Context,
	server *googlegrpc.Server,
	listener net.Listener,
	log *zap.Logger,
	serviceName string,
	shutdownTimeout time.Duration,
) error {
	if shutdownTimeout <= 0 {
		shutdownTimeout = DefaultShutdownTimeout
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info(serviceName+" started", zap.String("addr", listener.Addr().String()))
		errCh <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down " + serviceName)
		GracefulStop(server, shutdownTimeout, log)
		return nil
	case err := <-errCh:
		return err
	}
}

// GracefulStop останавливает gRPC-сервер с таймаутом и принудительным Stop при необходимости.
func GracefulStop(server *googlegrpc.Server, timeout time.Duration, log *zap.Logger) {
	if server == nil {
		return
	}
	if timeout <= 0 {
		timeout = DefaultShutdownTimeout
	}

	done := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		log.Warn("graceful shutdown timed out, forcing stop")
		server.Stop()
	}
}
