package domain

import "context"

// UserRepository хранит учётные записи пользователей.
type UserRepository interface {
	Save(ctx context.Context, user User) error
	GetByEmail(ctx context.Context, email string) (User, error)
}
