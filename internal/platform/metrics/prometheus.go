package metrics

import (
	"context"
	"errors"
	"net/http"
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

func init() {
	grpc_prometheus.EnableHandlingTimeHistogram()
}

// UnaryServerInterceptor записывает метрики Prometheus для unary RPC.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return grpc_prometheus.UnaryServerInterceptor
}

// RegisterGRPC регистрирует коллекторы метрик gRPC-сервера.
func RegisterGRPC(server *grpc.Server) {
	grpc_prometheus.Register(server)
}

// StartHTTPServer отдаёт метрики Prometheus на /metrics до отмены контекста.
func StartHTTPServer(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
