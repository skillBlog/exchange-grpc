package grpc

import (
	"context"
	"strings"

	"github.com/exchange-grpc/shared/sessionvalidation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type userIDContextKey struct{}
type userRolesContextKey struct{}

// ContextWithUserID сохраняет user_id в контексте.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey{}, userID)
}

// ContextWithRoles сохраняет роли пользователя в контексте.
func ContextWithRoles(ctx context.Context, roles []string) context.Context {
	return context.WithValue(ctx, userRolesContextKey{}, append([]string(nil), roles...))
}

// UserIDFromContext возвращает user_id, установленный interceptor'ом.
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey{}).(string)
	return userID, ok && strings.TrimSpace(userID) != ""
}

// RolesFromContext возвращает роли пользователя из контекста.
func RolesFromContext(ctx context.Context) []string {
	roles, ok := ctx.Value(userRolesContextKey{}).([]string)
	if !ok {
		return nil
	}
	return append([]string(nil), roles...)
}

// OutgoingContextWithAuth выпускает JWT и добавляет Bearer metadata в контекст.
func OutgoingContextWithAuth(ctx context.Context, tokens *sessionvalidation.TokenService, userID string, roles []string) (context.Context, error) {
	if tokens == nil {
		return ctx, status.Error(codes.Internal, "token service is not configured")
	}
	token, err := tokens.Issue(userID, roles)
	if err != nil {
		return ctx, err
	}
	return OutgoingContextWithBearer(ctx, token), nil
}

// ErrMissingUserID возвращается, когда user_id отсутствует в контексте.
func ErrMissingUserID() error {
	return status.Error(codes.Unauthenticated, "user_id is required")
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
