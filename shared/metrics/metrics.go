package metrics

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultShutdownTimeout = 2 * time.Second

var registerOnce sync.Once

var (
	started = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "grpc_server_started_total",
		Help: "Number of RPCs started on the server.",
	}, []string{"grpc_service", "grpc_method", "grpc_type"})

	handled = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "grpc_server_handled_total",
		Help: "Number of RPCs completed on the server, regardless of success or failure.",
	}, []string{"grpc_service", "grpc_method", "grpc_type", "grpc_code"})

	handling = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "grpc_server_handling_seconds",
		Help:    "Histogram of response latency (seconds) of gRPC that had been handled by the server.",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	}, []string{"grpc_service", "grpc_method", "grpc_type"})
)

func init() {
	register()
}

func register() {
	registerOnce.Do(func() {
		prometheus.MustRegister(started, handled, handling)
	})
}

// Handler отдаёт Prometheus text exposition format.
func Handler() http.Handler {
	register()
	return promhttp.Handler()
}

// Serve поднимает HTTP /metrics. Пустой addr отключает endpoint.
func Serve(addr string, log *zap.Logger) *http.Server {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil
	}
	if log == nil {
		log = zap.NewNop()
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", Handler())
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("metrics server stopped", zap.Error(err))
		}
	}()
	log.Info("metrics listening", zap.String("addr", addr), zap.String("path", "/metrics"))
	return srv
}

// Shutdown останавливает metrics HTTP-сервер. nil безопасен.
func Shutdown(srv *http.Server, log *zap.Logger) {
	if srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil && log != nil {
		log.Warn("metrics server shutdown", zap.Error(err))
	}
}

// UnaryServerInterceptor считает unary RPC: started/handled + latency.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	register()
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		service, method := splitMethod(info.FullMethod)
		rpcLabels := prometheus.Labels{
			"grpc_service": service,
			"grpc_method":  method,
			"grpc_type":    "unary",
		}
		started.With(rpcLabels).Inc()
		start := time.Now()

		resp, err := handler(ctx, req)

		observe(rpcLabels, time.Since(start), err)
		return resp, err
	}
}

// StreamServerInterceptor считает streaming RPC.
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	register()
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		service, method := splitMethod(info.FullMethod)
		rpcLabels := prometheus.Labels{
			"grpc_service": service,
			"grpc_method":  method,
			"grpc_type":    streamRPCType(info),
		}
		started.With(rpcLabels).Inc()
		start := time.Now()

		err := handler(srv, stream)

		observe(rpcLabels, time.Since(start), err)
		return err
	}
}

func observe(rpcLabels prometheus.Labels, d time.Duration, err error) {
	handling.With(rpcLabels).Observe(d.Seconds())
	handled.With(prometheus.Labels{
		"grpc_service": rpcLabels["grpc_service"],
		"grpc_method":  rpcLabels["grpc_method"],
		"grpc_type":    rpcLabels["grpc_type"],
		"grpc_code":    statusCode(err).String(),
	}).Inc()
}

func statusCode(err error) codes.Code {
	if err == nil {
		return codes.OK
	}
	if st, ok := status.FromError(err); ok {
		return st.Code()
	}
	return codes.Unknown
}

func streamRPCType(info *grpc.StreamServerInfo) string {
	if info.IsClientStream && info.IsServerStream {
		return "bidi_stream"
	}
	if info.IsClientStream {
		return "client_stream"
	}
	return "server_stream"
}

func splitMethod(fullMethod string) (service, method string) {
	fullMethod = strings.TrimPrefix(fullMethod, "/")
	i := strings.LastIndex(fullMethod, "/")
	if i < 0 {
		return "unknown", fullMethod
	}
	return fullMethod[:i], fullMethod[i+1:]
}
