package grpc

import (
	"context"

	"google.golang.org/grpc"
)

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

// ChainUnaryServer объединяет unary server interceptors в один interceptor.
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

// ChainStreamServer объединяет stream server interceptors в один interceptor.
func ChainStreamServer(interceptors ...grpc.StreamServerInterceptor) grpc.StreamServerInterceptor {
	switch len(interceptors) {
	case 0:
		return func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, stream)
		}
	case 1:
		return interceptors[0]
	default:
		return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			chained := handler
			for i := len(interceptors) - 1; i >= 0; i-- {
				current := interceptors[i]
				next := chained
				chained = func(currentSrv any, currentStream grpc.ServerStream) error {
					return current(currentSrv, currentStream, info, next)
				}
			}
			return chained(srv, stream)
		}
	}
}
