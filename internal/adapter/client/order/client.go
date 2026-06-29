package order

import (
	"context"

	"github.com/exchange-grpc/internal/platform/interceptor"
	"github.com/exchange-grpc/internal/platform/tracing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Dial открывает gRPC-соединение с OrderService и интерсептором x-request-id.
func Dial(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptor.UnaryClientRequestID),
	}
	dialOpts = append(dialOpts, tracing.ClientDialOptions()...)
	dialOpts = append(dialOpts, opts...)

	return grpc.NewClient(target, dialOpts...)
}
