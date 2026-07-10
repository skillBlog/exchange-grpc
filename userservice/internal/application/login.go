package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/exchange-grpc/userservice/internal/domain"
)

// LoginInput — параметры входа пользователя.
type LoginInput struct {
	Email    string
	Password string
}

// LoginOutput — результат успешного входа.
type LoginOutput struct {
	AccessToken  string
	RefreshToken string
}

// Login аутентифицирует пользователя и выпускает токены.
type Login struct {
	users        domain.UserRepository
	hasher       PasswordHasher
	accessTokens AccessTokenIssuer
	refreshTokens RefreshTokenManager
	limiter      LoginRateLimiter
}

// NewLogin создаёт use case Login.
func NewLogin(
	users domain.UserRepository,
	hasher PasswordHasher,
	accessTokens AccessTokenIssuer,
	refreshTokens RefreshTokenManager,
	limiter LoginRateLimiter,
) *Login {
	return &Login{
		users:         users,
		hasher:        hasher,
		accessTokens:  accessTokens,
		refreshTokens: refreshTokens,
		limiter:       limiter,
	}
}

// Execute проверяет пароль и возвращает access/refresh token.
func (uc *Login) Execute(ctx context.Context, input LoginInput) (LoginOutput, error) {
	email := NormalizeEmail(input.Email)
	password := strings.TrimSpace(input.Password)
	if email == "" || password == "" {
		return LoginOutput{}, fmt.Errorf("%w: email and password are required", domain.ErrInvalidArgument)
	}

	if uc.limiter != nil && !uc.limiter.Allow(email) {
		return LoginOutput{}, fmt.Errorf("%w: too many login attempts", domain.ErrRateLimited)
	}

	user, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		return LoginOutput{}, domain.ErrUnauthorized
	}

	if err := uc.hasher.Compare(user.PasswordHash, password); err != nil {
		return LoginOutput{}, domain.ErrUnauthorized
	}

	accessToken, err := uc.accessTokens.Issue(user.ID, user.RoleStrings())
	if err != nil {
		return LoginOutput{}, fmt.Errorf("issue access token: %w", err)
	}

	refreshToken, err := uc.refreshTokens.Issue(ctx, user.ID)
	if err != nil {
		return LoginOutput{}, fmt.Errorf("issue refresh token: %w", err)
	}

	return LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
