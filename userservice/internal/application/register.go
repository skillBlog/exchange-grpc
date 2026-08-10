package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/exchange-grpc/shared/roles"
	"github.com/exchange-grpc/shared/tracing"
	"github.com/exchange-grpc/userservice/internal/domain"
	"github.com/google/uuid"
)

// RegisterInput — параметры регистрации пользователя.
type RegisterInput struct {
	Email    string
	Password string
}

// RegisterOutput — результат регистрации с токенами (без лишнего Login).
type RegisterOutput struct {
	UserID       string
	AccessToken  string
	RefreshToken string
}

// Register создаёт нового пользователя и сразу выпускает токены.
type Register struct {
	users         domain.UserRepository
	hasher        PasswordHasher
	accessTokens  AccessTokenIssuer
	refreshTokens RefreshTokenManager
}

// NewRegister создаёт use case Register.
func NewRegister(
	users domain.UserRepository,
	hasher PasswordHasher,
	accessTokens AccessTokenIssuer,
	refreshTokens RefreshTokenManager,
) *Register {
	return &Register{
		users:         users,
		hasher:        hasher,
		accessTokens:  accessTokens,
		refreshTokens: refreshTokens,
	}
}

// Execute регистрирует пользователя с дефолтной ролью user и выдаёт access/refresh.
func (uc *Register) Execute(ctx context.Context, input RegisterInput) (out RegisterOutput, err error) {
	ctx, span := tracing.Start(ctx, "user.Register")
	defer tracing.End(span, &err)

	email := NormalizeEmail(input.Email)
	password := strings.TrimSpace(input.Password)
	if err = ValidateEmail(email); err != nil {
		return RegisterOutput{}, err
	}
	if err = ValidatePassword(password); err != nil {
		return RegisterOutput{}, err
	}

	hash, err := uc.hasher.Hash(password)
	if err != nil {
		return RegisterOutput{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := domain.NewUser(uuid.NewString(), email, hash, []roles.Role{roles.RoleUser})
	if err != nil {
		return RegisterOutput{}, err
	}

	if err = uc.users.Save(ctx, user); err != nil {
		return RegisterOutput{}, err
	}

	accessToken, err := uc.accessTokens.Issue(user.ID, user.RoleStrings())
	if err != nil {
		return RegisterOutput{}, fmt.Errorf("issue access token: %w", err)
	}

	refreshToken, err := uc.refreshTokens.Issue(ctx, user.ID)
	if err != nil {
		return RegisterOutput{}, fmt.Errorf("issue refresh token: %w", err)
	}

	return RegisterOutput{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
