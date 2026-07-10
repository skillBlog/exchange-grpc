package application

import "context"

// PasswordHasher хеширует и сверяет пароли.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// AccessTokenIssuer выпускает JWT access token.
type AccessTokenIssuer interface {
	Issue(userID string, roles []string) (string, error)
}

// RefreshTokenManager управляет opaque refresh token.
type RefreshTokenManager interface {
	Issue(ctx context.Context, userID string) (string, error)
	Validate(ctx context.Context, token string) (userID string, err error)
	Revoke(ctx context.Context, token string) error
}

// LoginRateLimiter ограничивает частоту попыток входа по email.
type LoginRateLimiter interface {
	Allow(email string) bool
}
