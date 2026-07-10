package grpc

import (
	"context"
	"strings"

	"github.com/exchange-grpc/shared/roles"
	"github.com/exchange-grpc/shared/sessionvalidation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	// MetadataAuthorization — стандартный заголовок Bearer JWT.
	MetadataAuthorization = "authorization"
	bearerPrefix          = "Bearer "
)

// NewUnaryServerJWTAuth проверяет JWT и кладёт user_id/roles в контекст.
// publicMethods — полные имена RPC без обязательной авторизации.
func NewUnaryServerJWTAuth(tokens *sessionvalidation.TokenService, publicMethods ...string) grpc.UnaryServerInterceptor {
	public := make(map[string]struct{}, len(publicMethods))
	for _, method := range publicMethods {
		public[method] = struct{}{}
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if _, skip := public[info.FullMethod]; skip {
			return handler(ctx, req)
		}

		enriched, err := enrichContextFromJWT(ctx, tokens)
		if err != nil {
			return nil, err
		}
		return handler(enriched, req)
	}
}

// NewStreamServerJWTAuth проверяет JWT для server-streaming RPC.
func NewStreamServerJWTAuth(tokens *sessionvalidation.TokenService) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		enriched, err := enrichContextFromJWT(stream.Context(), tokens)
		if err != nil {
			return err
		}
		return handler(srv, &wrappedServerStream{ServerStream: stream, ctx: enriched})
	}
}

func enrichContextFromJWT(ctx context.Context, tokens *sessionvalidation.TokenService) (context.Context, error) {
	if tokens == nil {
		return ctx, status.Error(codes.Internal, "token service is not configured")
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, status.Error(codes.Unauthenticated, "authorization is required")
	}

	values := md.Get(MetadataAuthorization)
	if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
		return ctx, status.Error(codes.Unauthenticated, "authorization is required")
	}

	raw := strings.TrimSpace(values[0])
	if !strings.HasPrefix(raw, bearerPrefix) {
		return ctx, status.Error(codes.Unauthenticated, "authorization must use Bearer scheme")
	}

	claims, err := tokens.Validate(strings.TrimPrefix(raw, bearerPrefix))
	if err != nil {
		return ctx, status.Error(codes.Unauthenticated, "invalid or expired token")
	}

	ctx = ContextWithUserID(ctx, claims.UserID)
	ctx = ContextWithRoles(ctx, roles.NormalizeStrings(claims.Roles))
	return ctx, nil
}

// OutgoingContextWithBearer добавляет Bearer JWT в исходящий gRPC metadata.
func OutgoingContextWithBearer(ctx context.Context, accessToken string) context.Context {
	md := metadata.Pairs(MetadataAuthorization, bearerPrefix+strings.TrimSpace(accessToken))
	if existing, ok := metadata.FromOutgoingContext(ctx); ok {
		md = metadata.Join(existing, md)
	}
	return metadata.NewOutgoingContext(ctx, md)
}

// UnaryClientForwardAuthorization пробрасывает Bearer JWT из incoming metadata в исходящий вызов.
func UnaryClientForwardAuthorization(
	ctx context.Context,
	method string,
	req, reply any,
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		if values := md.Get(MetadataAuthorization); len(values) > 0 && strings.TrimSpace(values[0]) != "" {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
	}

	if inMD, ok := metadata.FromIncomingContext(ctx); ok {
		if values := inMD.Get(MetadataAuthorization); len(values) > 0 && strings.TrimSpace(values[0]) != "" {
			outMD := metadata.Pairs(MetadataAuthorization, values[0])
			if existing, ok := metadata.FromOutgoingContext(ctx); ok {
				outMD = metadata.Join(existing, outMD)
			}
			ctx = metadata.NewOutgoingContext(ctx, outMD)
		}
	}

	return invoker(ctx, method, req, reply, cc, opts...)
}
