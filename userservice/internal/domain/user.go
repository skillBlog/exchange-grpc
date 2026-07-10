package domain

import (
	"fmt"
	"strings"

	"github.com/exchange-grpc/shared/roles"
)

// User — учётная запись пользователя биржи.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Roles        []roles.Role
}

// NewUser создаёт пользователя с валидацией email, пароля и ролей.
func NewUser(id, email, passwordHash string, userRoles []roles.Role) (User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return User{}, fmt.Errorf("%w: email is required", ErrInvalidArgument)
	}
	if strings.TrimSpace(passwordHash) == "" {
		return User{}, fmt.Errorf("%w: password hash is required", ErrInvalidArgument)
	}

	if len(userRoles) == 0 {
		userRoles = []roles.Role{roles.RoleUser}
	}

	normalized := make([]roles.Role, 0, len(userRoles))
	for _, role := range userRoles {
		parsed, ok := roles.Parse(string(role))
		if !ok {
			return User{}, fmt.Errorf("%w: invalid role %q", ErrInvalidArgument, role)
		}
		normalized = append(normalized, parsed)
	}

	return User{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		Roles:        normalized,
	}, nil
}

// RoleStrings возвращает роли в виде строк для JWT и API.
func (u User) RoleStrings() []string {
	result := make([]string, len(u.Roles))
	for i, role := range u.Roles {
		result[i] = string(role)
	}
	return result
}
