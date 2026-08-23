package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/exchange-grpc/shared/tracing"
	"github.com/exchange-grpc/userservice/internal/domain"
)

// RefreshTokenInput — параметры обновления access token.
type RefreshTokenInput struct {
	RefreshToken string
}

// RefreshTokenOutput — новая пара access/refresh (rotation).
type RefreshTokenOutput struct {
	AccessToken  string
	RefreshToken string
}

// RefreshToken обновляет access token и ротирует refresh token.
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

// Execute выпускает новый access token и новый refresh token, отзывая старый.
func (uc *RefreshToken) Execute(ctx context.Context, input RefreshTokenInput) (out RefreshTokenOutput, err error) {
	ctx, span := tracing.Start(ctx, "user.RefreshToken")
	defer tracing.End(span, &err)

	oldRefresh := strings.TrimSpace(input.RefreshToken)
	if oldRefresh == "" {
		return RefreshTokenOutput{}, fmt.Errorf("%w: refresh token is required", domain.ErrInvalidArgument)
	}

	userID, err := uc.refreshTokens.Validate(ctx, oldRefresh)
	if err != nil {
		return RefreshTokenOutput{}, err
	}

	// Актуальные роли только из БД: opaque refresh хранит user_id, не claims.
	user, err := uc.users.GetByID(ctx, userID)
	if err != nil {
		return RefreshTokenOutput{}, domain.ErrUnauthorized
	}

	accessToken, err := uc.accessTokens.Issue(user.ID, user.RoleStrings())
	if err != nil {
		return RefreshTokenOutput{}, fmt.Errorf("issue access token: %w", err)
	}

	newRefresh, err := uc.refreshTokens.Rotate(ctx, oldRefresh, user.ID)
	if err != nil {
		return RefreshTokenOutput{}, fmt.Errorf("rotate refresh token: %w", err)
	}

	return RefreshTokenOutput{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
	}, nil
}
