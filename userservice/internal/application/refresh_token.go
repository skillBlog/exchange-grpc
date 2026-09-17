package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/exchange-grpc/shared/tracing"
	"github.com/exchange-grpc/userservice/internal/domain"
	"go.uber.org/zap"
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
	log           *zap.Logger
}

// NewRefreshToken создаёт use case RefreshToken.
func NewRefreshToken(
	users domain.UserRepository,
	accessTokens AccessTokenIssuer,
	refreshTokens RefreshTokenManager,
	log *zap.Logger,
) *RefreshToken {
	if log == nil {
		log = zap.NewNop()
	}
	return &RefreshToken{
		users:         users,
		accessTokens:  accessTokens,
		refreshTokens: refreshTokens,
		log:           log,
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
		return RefreshTokenOutput{}, fmt.Errorf("validate refresh token: %w", err)
	}

	// Актуальные роли только из БД: opaque refresh хранит user_id, не claims.
	user, err := uc.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return RefreshTokenOutput{}, fmt.Errorf("get user: %w", domain.ErrUnauthorized)
		}
		return RefreshTokenOutput{}, fmt.Errorf("get user: %w", err)
	}
	span.SetAttributes(tracing.Attr("user_id", user.ID))

	accessToken, err := uc.accessTokens.Issue(user.ID, user.RoleStrings())
	if err != nil {
		return RefreshTokenOutput{}, fmt.Errorf("issue access token: %w", err)
	}

	newRefresh, err := uc.refreshTokens.Rotate(ctx, oldRefresh, user.ID)
	if err != nil {
		return RefreshTokenOutput{}, fmt.Errorf("rotate refresh token: %w", err)
	}

	logAudit(ctx, uc.log, "refresh token rotated", user.ID)

	return RefreshTokenOutput{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
	}, nil
}
