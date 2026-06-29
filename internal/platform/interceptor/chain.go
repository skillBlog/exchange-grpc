package interceptor

import (
	"context"

	"google.golang.org/grpc"
)

// ChainUnaryServer объединяет unary server interceptors в один interceptor.
// Interceptors выполняются в порядке передачи: первый — самый внешний.
func ChainUnaryServer(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	switch len(interceptors) {
	case 0:
		return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			return handler(ctx, req)
		}
	case 1:
		return interceptors[0]
	default:
		return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			chained := handler
			for i := len(interceptors) - 1; i >= 0; i-- {
				current := interceptors[i]
				next := chained
				chained = func(currentCtx context.Context, currentReq any) (any, error) {
					return current(currentCtx, currentReq, info, next)
				}
			}
			return chained(ctx, req)
		}
	}
}

// ChainUnaryClient объединяет unary client interceptors в один interceptor.
func ChainUnaryClient(interceptors ...grpc.UnaryClientInterceptor) grpc.UnaryClientInterceptor {
	switch len(interceptors) {
	case 0:
		return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
	case 1:
		return interceptors[0]
	default:
		return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			chained := invoker
			for i := len(interceptors) - 1; i >= 0; i-- {
				current := interceptors[i]
				next := chained
				chained = func(currentCtx context.Context, currentMethod string, currentReq, currentReply any, currentCC *grpc.ClientConn, currentOpts ...grpc.CallOption) error {
					return current(currentCtx, currentMethod, currentReq, currentReply, currentCC, next, currentOpts...)
				}
			}
			return chained(ctx, method, req, reply, cc, opts...)
		}
	}
}
