package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/exchange-grpc/userservice/internal/domain"
)

// GetUserInput — параметры получения профиля.
type GetUserInput struct {
	UserID string
}

// GetUserOutput — публичный профиль пользователя.
type GetUserOutput struct {
	UserID string
	Email  string
	Roles  []string
}

// GetUser возвращает профиль пользователя без чувствительных данных.
type GetUser struct {
	users domain.UserRepository
}

// NewGetUser создаёт use case GetUser.
func NewGetUser(users domain.UserRepository) *GetUser {
	return &GetUser{users: users}
}

// Execute возвращает профиль по user_id из JWT.
func (uc *GetUser) Execute(ctx context.Context, input GetUserInput) (GetUserOutput, error) {
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return GetUserOutput{}, fmt.Errorf("%w: user_id is required", domain.ErrInvalidArgument)
	}

	user, err := uc.users.GetByID(ctx, userID)
	if err != nil {
		return GetUserOutput{}, err
	}

	return GetUserOutput{
		UserID: user.ID,
		Email:  user.Email,
		Roles:  user.RoleStrings(),
	}, nil
}
