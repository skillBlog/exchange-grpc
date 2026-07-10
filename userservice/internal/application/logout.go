package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/exchange-grpc/userservice/internal/domain"
)

// LogoutInput — параметры выхода.
type LogoutInput struct {
	RefreshToken string
}

// Logout отзывает refresh token.
type Logout struct {
	refreshTokens RefreshTokenManager
}

// NewLogout создаёт use case Logout.
func NewLogout(refreshTokens RefreshTokenManager) *Logout {
	return &Logout{refreshTokens: refreshTokens}
}

// Execute инвалидирует refresh token.
func (uc *Logout) Execute(ctx context.Context, input LogoutInput) error {
	token := strings.TrimSpace(input.RefreshToken)
	if token == "" {
		return fmt.Errorf("%w: refresh token is required", domain.ErrInvalidArgument)
	}
	return uc.refreshTokens.Revoke(ctx, token)
}
