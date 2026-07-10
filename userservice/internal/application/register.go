package application

import (
	"context"
	"fmt"

	"github.com/exchange-grpc/shared/roles"
	"github.com/exchange-grpc/userservice/internal/domain"
	"github.com/google/uuid"
)

// RegisterInput — параметры регистрации пользователя.
type RegisterInput struct {
	Email    string
	Password string
}

// RegisterOutput — результат регистрации.
type RegisterOutput struct {
	UserID string
}

// Register создаёт нового пользователя.
type Register struct {
	users  domain.UserRepository
	hasher PasswordHasher
}

// NewRegister создаёт use case Register.
func NewRegister(users domain.UserRepository, hasher PasswordHasher) *Register {
	return &Register{users: users, hasher: hasher}
}

// Execute регистрирует пользователя с дефолтной ролью user.
func (uc *Register) Execute(ctx context.Context, input RegisterInput) (RegisterOutput, error) {
	email := NormalizeEmail(input.Email)
	if err := ValidateEmail(email); err != nil {
		return RegisterOutput{}, err
	}
	if err := ValidatePassword(input.Password); err != nil {
		return RegisterOutput{}, err
	}

	hash, err := uc.hasher.Hash(input.Password)
	if err != nil {
		return RegisterOutput{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := domain.NewUser(uuid.NewString(), email, hash, []roles.Role{roles.RoleUser})
	if err != nil {
		return RegisterOutput{}, err
	}

	if err := uc.users.Save(ctx, user); err != nil {
		return RegisterOutput{}, err
	}

	return RegisterOutput{UserID: user.ID}, nil
}
