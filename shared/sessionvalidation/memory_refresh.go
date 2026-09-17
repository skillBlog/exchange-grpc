package sessionvalidation

import (
	"context"
	"sync"
	"time"

	sharederrors "github.com/exchange-grpc/shared/errors"
)

// MemoryRefreshTokenStore — in-memory реализация RefreshTokenStore для тестов.
type MemoryRefreshTokenStore struct {
	mu     sync.RWMutex
	byID   map[string]RefreshToken
	byHash map[string]string
}

// NewMemoryRefreshTokenStore создаёт пустое in-memory хранилище refresh token.
func NewMemoryRefreshTokenStore() *MemoryRefreshTokenStore {
	return &MemoryRefreshTokenStore{
		byID:   make(map[string]RefreshToken),
		byHash: make(map[string]string),
	}
}

// Save сохраняет refresh token.
func (s *MemoryRefreshTokenStore) Save(ctx context.Context, token RefreshToken) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.byID[token.ID] = token
	s.byHash[token.TokenHash] = token.ID
	return nil
}

// GetByTokenHash возвращает refresh token по хешу.
func (s *MemoryRefreshTokenStore) GetByTokenHash(ctx context.Context, tokenHash string) (RefreshToken, error) {
	if err := ctx.Err(); err != nil {
		return RefreshToken{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.byHash[tokenHash]
	if !ok {
		return RefreshToken{}, sharederrors.ErrUnauthorized
	}
	token, ok := s.byID[id]
	if !ok {
		return RefreshToken{}, sharederrors.ErrUnauthorized
	}
	return token, nil
}

// Revoke помечает refresh token отозванным.
func (s *MemoryRefreshTokenStore) Revoke(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	token, ok := s.byID[id]
	if !ok || token.RevokedAt != nil {
		return nil
	}
	now := time.Now()
	token.RevokedAt = &now
	s.byID[id] = token
	return nil
}

// Rotate атомарно отзывает старый refresh token и сохраняет новый.
func (s *MemoryRefreshTokenStore) Rotate(ctx context.Context, oldTokenHash string, now time.Time, newToken RefreshToken) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id, ok := s.byHash[oldTokenHash]
	if !ok {
		return sharederrors.ErrUnauthorized
	}
	old, ok := s.byID[id]
	if !ok || !old.IsActive(now) || old.UserID != newToken.UserID {
		return sharederrors.ErrUnauthorized
	}

	revokedAt := now
	old.RevokedAt = &revokedAt
	s.byID[id] = old
	s.byID[newToken.ID] = newToken
	s.byHash[newToken.TokenHash] = newToken.ID
	return nil
}
