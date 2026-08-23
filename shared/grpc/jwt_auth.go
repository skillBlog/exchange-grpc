package grpc

import (
	"context"
	"strings"
	"sync"

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
	bearerPrefix          = "bearer "
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

// NewStreamServerJWTAuth проверяет JWT для streaming RPC.
// publicMethods — полные имена RPC без обязательной авторизации (например Health/Watch).
func NewStreamServerJWTAuth(tokens *sessionvalidation.TokenService, publicMethods ...string) grpc.StreamServerInterceptor {
	public := make(map[string]struct{}, len(publicMethods))
	for _, method := range publicMethods {
		public[method] = struct{}{}
	}

	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if _, skip := public[info.FullMethod]; skip {
			return handler(srv, stream)
		}
		// JWT после RecvMsg, чтобы proto-validate успел отсечь невалидный запрос.
		return handler(srv, &jwtAuthServerStream{ServerStream: stream, tokens: tokens})
	}
}

type jwtAuthServerStream struct {
	grpc.ServerStream
	tokens  *sessionvalidation.TokenService
	ctx     context.Context
	once    sync.Once
	authErr error
}

func (s *jwtAuthServerStream) RecvMsg(m any) error {
	if err := s.ServerStream.RecvMsg(m); err != nil {
		return err
	}
	s.once.Do(func() {
		s.ctx, s.authErr = enrichContextFromJWT(s.ServerStream.Context(), s.tokens)
	})
	return s.authErr
}

func (s *jwtAuthServerStream) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return s.ServerStream.Context()
}

func enrichContextFromJWT(ctx context.Context, tokens *sessionvalidation.TokenService) (context.Context, error) {
	if tokens == nil {
		return ctx, status.Error(codes.Internal, "token service is not configured")
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, status.Error(codes.Unauthenticated, "authorization is required")
	}

	raw, ok := firstAuthorizationValue(md)
	if !ok {
		return ctx, status.Error(codes.Unauthenticated, "authorization is required")
	}

	token, ok := parseBearerToken(raw)
	if !ok {
		return ctx, status.Error(codes.Unauthenticated, "authorization must use Bearer scheme")
	}

	claims, err := tokens.Validate(token)
	if err != nil {
		return ctx, status.Error(codes.Unauthenticated, "invalid or expired token")
	}

	ctx = ContextWithUserID(ctx, claims.UserID)
	ctx = ContextWithRoles(ctx, roles.NormalizeStrings(claims.Roles))
	return ctx, nil
}

// firstAuthorizationValue читает authorization metadata без зависимости от регистра ключа.
func firstAuthorizationValue(md metadata.MD) (string, bool) {
	if md == nil {
		return "", false
	}
	for key, values := range md {
		if !strings.EqualFold(strings.TrimSpace(key), MetadataAuthorization) {
			continue
		}
		for _, value := range values {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				return trimmed, true
			}
		}
	}
	return "", false
}

// parseBearerToken принимает Bearer/bearer/BEARER и возвращает JWT.
func parseBearerToken(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if len(raw) < len(bearerPrefix) {
		return "", false
	}
	if !strings.EqualFold(raw[:len(bearerPrefix)], bearerPrefix) {
		return "", false
	}
	token := strings.TrimSpace(raw[len(bearerPrefix):])
	return token, token != ""
}

// OutgoingContextWithBearer добавляет Bearer JWT в исходящий gRPC metadata.
// Использует Set, чтобы не дублировать authorization через Join.
func OutgoingContextWithBearer(ctx context.Context, accessToken string) context.Context {
	token := strings.TrimSpace(accessToken)
	md, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		md = md.Copy()
	} else {
		md = metadata.MD{}
	}
	md.Set(MetadataAuthorization, "Bearer "+token)
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
		if _, found := firstAuthorizationValue(md); found {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
	}

	if inMD, ok := metadata.FromIncomingContext(ctx); ok {
		if raw, found := firstAuthorizationValue(inMD); found {
			if token, ok := parseBearerToken(raw); ok {
				ctx = OutgoingContextWithBearer(ctx, token)
			} else {
				md, hasOutgoing := metadata.FromOutgoingContext(ctx)
				if hasOutgoing {
					md = md.Copy()
				} else {
					md = metadata.MD{}
				}
				md.Set(MetadataAuthorization, raw)
				ctx = metadata.NewOutgoingContext(ctx, md)
			}
		}
	}

	return invoker(ctx, method, req, reply, cc, opts...)
}
