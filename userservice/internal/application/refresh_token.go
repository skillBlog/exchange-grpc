package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/exchange-grpc/userservice/internal/domain"
)

// RefreshTokenInput — параметры обновления access token.
type RefreshTokenInput struct {
	RefreshToken string
}

// RefreshTokenOutput — новый access token.
type RefreshTokenOutput struct {
	AccessToken string
}

// RefreshToken обновляет access token по refresh token.
type RefreshToken struct {
	users         domain.UserRepository
	accessTokens  AccessTokenIssuer
	refreshTokens RefreshTokenManager
}

// NewRefreshToken создаёт use case RefreshToken.
func NewRefreshToken(
	users domain.UserRepository,
	accessTokens AccessTokenIssuer,
	refreshTokens RefreshTokenManager,
) *RefreshToken {
	return &RefreshToken{
		users:         users,
		accessTokens:  accessTokens,
		refreshTokens: refreshTokens,
	}
}

// Execute выпускает новый access token.
func (uc *RefreshToken) Execute(ctx context.Context, input RefreshTokenInput) (RefreshTokenOutput, error) {
	token := strings.TrimSpace(input.RefreshToken)
	if token == "" {
		return RefreshTokenOutput{}, fmt.Errorf("%w: refresh token is required", domain.ErrInvalidArgument)
	}

	userID, err := uc.refreshTokens.Validate(ctx, token)
	if err != nil {
		return RefreshTokenOutput{}, err
	}

	user, err := uc.users.GetByID(ctx, userID)
	if err != nil {
		return RefreshTokenOutput{}, domain.ErrUnauthorized
	}

	accessToken, err := uc.accessTokens.Issue(user.ID, user.RoleStrings())
	if err != nil {
		return RefreshTokenOutput{}, fmt.Errorf("issue access token: %w", err)
	}

	return RefreshTokenOutput{AccessToken: accessToken}, nil
}
