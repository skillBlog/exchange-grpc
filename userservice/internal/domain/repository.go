package domain

import (
	"context"
	"time"
)

// UserRepository хранит учётные записи пользователей.
type UserRepository interface {
	Save(ctx context.Context, user User) error
	GetByEmail(ctx context.Context, email string) (User, error)
	GetByID(ctx context.Context, id string) (User, error)
}

// RefreshTokenRepository хранит refresh token.
type RefreshTokenRepository interface {
	Save(ctx context.Context, token RefreshToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (RefreshToken, error)
	Revoke(ctx context.Context, id string) error
	// Rotate атомарно отзывает токен с oldTokenHash и сохраняет newToken.
	// Возвращает ErrUnauthorized, если старый токен отсутствует, просрочен,
	// уже отозван или принадлежит другому пользователю.
	Rotate(ctx context.Context, oldTokenHash string, now time.Time, newToken RefreshToken) error
}
