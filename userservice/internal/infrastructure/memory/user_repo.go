package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/exchange-grpc/userservice/internal/domain"
)

// UserRepository — in-memory хранилище пользователей.
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
		return domain.User{}, fmt.Errorf("%w: user not found", domain.ErrUnauthorized)
	}

	user, ok := r.byID[userID]
	if !ok {
		return domain.User{}, fmt.Errorf("%w: user not found", domain.ErrUnauthorized)
	}
	return user, nil
}

func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}
