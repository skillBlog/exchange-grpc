package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/exchange-grpc/userservice/internal/domain"
)

// UserRepository — in-memory хранилище пользователей для тестов.
type UserRepository struct {
	mu    sync.RWMutex
	byID  map[string]domain.User
	email map[string]string
}

// NewUserRepository создаёт пустой in-memory репозиторий.
func NewUserRepository() *UserRepository {
	return &UserRepository{
		byID:  make(map[string]domain.User),
		email: make(map[string]string),
	}
}

// Save сохраняет пользователя.
func (r *UserRepository) Save(ctx context.Context, user domain.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	normalizedEmail := normalizeEmail(user.Email)
	if existingID, exists := r.email[normalizedEmail]; exists && existingID != user.ID {
		return fmt.Errorf("%w: email already registered", domain.ErrAlreadyExists)
	}

	r.byID[user.ID] = user
	r.email[normalizedEmail] = user.ID
	return nil
}

// GetByEmail возвращает пользователя по email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	userID, ok := r.email[normalizeEmail(email)]
	if !ok {
		return domain.User{}, domain.ErrUnauthorized
	}
	return r.byID[userID], nil
}

// GetByID возвращает пользователя по идентификатору.
func (r *UserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.byID[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return user, nil
}

func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

// RefreshTokenRepository — in-memory хранилище refresh token для тестов.
type RefreshTokenRepository struct {
	mu     sync.RWMutex
	byID   map[string]domain.RefreshToken
	byHash map[string]string
}

// NewRefreshTokenRepository создаёт in-memory refresh token repository.
func NewRefreshTokenRepository() *RefreshTokenRepository {
	return &RefreshTokenRepository{
		byID:   make(map[string]domain.RefreshToken),
		byHash: make(map[string]string),
	}
}

// Save сохраняет refresh token.
func (r *RefreshTokenRepository) Save(ctx context.Context, token domain.RefreshToken) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.byID[token.ID] = token
	r.byHash[token.TokenHash] = token.ID
	return nil
}

// GetByTokenHash возвращает refresh token по хешу.
func (r *RefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (domain.RefreshToken, error) {
	if err := ctx.Err(); err != nil {
		return domain.RefreshToken{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.byHash[tokenHash]
	if !ok {
		return domain.RefreshToken{}, domain.ErrUnauthorized
	}
	token, ok := r.byID[id]
	if !ok {
		return domain.RefreshToken{}, domain.ErrUnauthorized
	}
	return token, nil
}

// Revoke помечает refresh token отозванным.
func (r *RefreshTokenRepository) Revoke(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	token, ok := r.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	now := time.Now()
	token.RevokedAt = &now
	r.byID[id] = token
	return nil
}
