package domain

import (
	"fmt"
	"strings"
)

// User — учётная запись пользователя биржи.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Roles        []string
}

// NewUser создаёт пользователя с валидацией email и пароля.
func NewUser(id, email, passwordHash string, roles []string) (User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return User{}, fmt.Errorf("%w: email is required", ErrInvalidArgument)
	}
	if strings.TrimSpace(passwordHash) == "" {
		return User{}, fmt.Errorf("%w: password hash is required", ErrInvalidArgument)
	}

	return User{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		Roles:        append([]string(nil), roles...),
	}, nil
}
