package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/exchange-grpc/shared/sessionvalidation"
	"github.com/exchange-grpc/userservice/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

// LoginInput — параметры входа пользователя.
type LoginInput struct {
	Email    string
	Password string
}

// LoginOutput — результат успешного входа.
type LoginOutput struct {
	AccessToken string
}

// Login аутентифицирует пользователя и выпускает JWT.
type Login struct {
	users  domain.UserRepository
	tokens *sessionvalidation.TokenService
}

// NewLogin создаёт use case Login.
func NewLogin(users domain.UserRepository, tokens *sessionvalidation.TokenService) *Login {
	return &Login{users: users, tokens: tokens}
}

// Execute проверяет пароль и возвращает access token.
func (uc *Login) Execute(ctx context.Context, input LoginInput) (LoginOutput, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))
	password := strings.TrimSpace(input.Password)
	if email == "" || password == "" {
		return LoginOutput{}, fmt.Errorf("%w: email and password are required", domain.ErrInvalidArgument)
	}

	user, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		return LoginOutput{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return LoginOutput{}, fmt.Errorf("%w: invalid credentials", domain.ErrUnauthorized)
	}

	token, err := uc.tokens.Issue(user.ID, user.Roles)
	if err != nil {
		return LoginOutput{}, fmt.Errorf("issue token: %w", err)
	}

	return LoginOutput{AccessToken: token}, nil
}
