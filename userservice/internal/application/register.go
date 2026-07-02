package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/exchange-grpc/userservice/internal/domain"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLength = 8
	bcryptCost        = 12
)

// RegisterInput — параметры регистрации пользователя.
type RegisterInput struct {
	Email    string
	Password string
	Roles    []string
}

// RegisterOutput — результат регистрации.
type RegisterOutput struct {
	UserID string
}

// Register создаёт нового пользователя.
type Register struct {
	users domain.UserRepository
}

// NewRegister создаёт use case Register.
func NewRegister(users domain.UserRepository) *Register {
	return &Register{users: users}
}

// Execute регистрирует пользователя с хешированием пароля.
func (uc *Register) Execute(ctx context.Context, input RegisterInput) (RegisterOutput, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))
	password := strings.TrimSpace(input.Password)
	if email == "" {
		return RegisterOutput{}, fmt.Errorf("%w: email is required", domain.ErrInvalidArgument)
	}
	if len(password) < minPasswordLength {
		return RegisterOutput{}, fmt.Errorf("%w: password must be at least %d characters", domain.ErrInvalidArgument, minPasswordLength)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return RegisterOutput{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := domain.NewUser(uuid.NewString(), email, string(hash), input.Roles)
	if err != nil {
		return RegisterOutput{}, err
	}

	if err := uc.users.Save(ctx, user); err != nil {
		return RegisterOutput{}, err
	}

	return RegisterOutput{UserID: user.ID}, nil
}
