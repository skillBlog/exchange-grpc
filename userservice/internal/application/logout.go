package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/exchange-grpc/shared/tracing"
	"github.com/exchange-grpc/userservice/internal/domain"
	"go.uber.org/zap"
)

// LogoutInput — параметры выхода.
type LogoutInput struct {
	RefreshToken string
}

// Logout отзывает refresh token.
type Logout struct {
	refreshTokens RefreshTokenManager
	log           *zap.Logger
}

// NewLogout создаёт use case Logout.
func NewLogout(refreshTokens RefreshTokenManager, log *zap.Logger) *Logout {
	if log == nil {
		log = zap.NewNop()
	}
	return &Logout{refreshTokens: refreshTokens, log: log}
}

// Execute инвалидирует refresh token.
func (uc *Logout) Execute(ctx context.Context, input LogoutInput) (err error) {
	ctx, span := tracing.Start(ctx, "user.Logout")
	defer tracing.End(span, &err)

	token := strings.TrimSpace(input.RefreshToken)
	if token == "" {
		return fmt.Errorf("%w: refresh token is required", domain.ErrInvalidArgument)
	}
	if err = uc.refreshTokens.Revoke(ctx, token); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	logAudit(ctx, uc.log, "user logged out", "")
	return nil
}
