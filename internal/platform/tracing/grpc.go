package tracing

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

// ServerOptions возвращает опции gRPC-сервера для распределённой трассировки.
func ServerOptions() []grpc.ServerOption {
	if !Enabled() {
		return nil
	}
	return []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	}
}

// ClientDialOptions возвращает опции gRPC-клиента для распределённой трассировки.
func ClientDialOptions() []grpc.DialOption {
	if !Enabled() {
		return nil
	}
	return []grpc.DialOption{
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}
}
