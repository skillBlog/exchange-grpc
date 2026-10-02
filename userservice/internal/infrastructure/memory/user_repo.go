package memory

import (
	"context"
	"fmt"
	"sync"

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

// Create вставляет нового пользователя. Повтор с тем же id заменяет запись — удобно в тестах.
func (r *UserRepository) Create(ctx context.Context, user domain.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	normalizedEmail := domain.NormalizeEmail(user.Email)
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

	userID, ok := r.email[domain.NormalizeEmail(email)]
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
