package sessionvalidation

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	sharederrors "github.com/exchange-grpc/shared/errors"
	"github.com/google/uuid"
)

// RefreshToken — сохранённый opaque refresh token.
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// IsActive сообщает, можно ли использовать refresh token.
func (t RefreshToken) IsActive(now time.Time) bool {
	if t.RevokedAt != nil {
		return false
	}
	return now.Before(t.ExpiresAt)
}

// RefreshTokenStore — хранилище hashed refresh token.
// Реализации: Postgres в userservice, memory для тестов. Redis — отдельный шаг.
type RefreshTokenStore interface {
	Save(ctx context.Context, token RefreshToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (RefreshToken, error)
	Revoke(ctx context.Context, id string) error
	Rotate(ctx context.Context, oldTokenHash string, now time.Time, newToken RefreshToken) error
}

// RefreshTokenService выпускает непрозрачные refresh token и работает через Store.
type RefreshTokenService struct {
	store RefreshTokenStore
	ttl   time.Duration
	now   func() time.Time
}

// NewRefreshTokenService создаёт сервис refresh token.
func NewRefreshTokenService(store RefreshTokenStore, ttl time.Duration) *RefreshTokenService {
	return &RefreshTokenService{
		store: store,
		ttl:   ttl,
		now:   time.Now,
	}
}

// Issue создаёт и сохраняет refresh token.
func (s *RefreshTokenService) Issue(ctx context.Context, userID string) (string, error) {
	raw, err := randomRefreshToken()
	if err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}

	now := s.now()
	token := RefreshToken{
		ID:        uuid.NewString(),
		UserID:    userID,
		TokenHash: hashRefreshToken(raw),
		ExpiresAt: now.Add(s.ttl),
	}
	if err := s.store.Save(ctx, token); err != nil {
		return "", err
	}
	return raw, nil
}

// Validate проверяет refresh token и возвращает user_id.
func (s *RefreshTokenService) Validate(ctx context.Context, raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("%w: refresh token is required", sharederrors.ErrUnauthorized)
	}

	stored, err := s.store.GetByTokenHash(ctx, hashRefreshToken(raw))
	if err != nil {
		return "", err
	}
	if !stored.IsActive(s.now()) {
		return "", fmt.Errorf("%w: refresh token expired or revoked", sharederrors.ErrUnauthorized)
	}
	return stored.UserID, nil
}

// Revoke отзывает refresh token.
// Идемпотентно: неизвестный или уже отозванный токен не считается ошибкой.
func (s *RefreshTokenService) Revoke(ctx context.Context, raw string) error {
	if raw == "" {
		return fmt.Errorf("%w: refresh token is required", sharederrors.ErrInvalidArgument)
	}

	stored, err := s.store.GetByTokenHash(ctx, hashRefreshToken(raw))
	if err != nil {
		if errors.Is(err, sharederrors.ErrUnauthorized) || errors.Is(err, sharederrors.ErrNotFound) {
			return nil
		}
		return err
	}
	return s.store.Revoke(ctx, stored.ID)
}

// Rotate атомарно отзывает старый refresh token и выдаёт новый.
func (s *RefreshTokenService) Rotate(ctx context.Context, oldRaw, userID string) (string, error) {
	oldRaw = strings.TrimSpace(oldRaw)
	userID = strings.TrimSpace(userID)
	if oldRaw == "" {
		return "", fmt.Errorf("%w: refresh token is required", sharederrors.ErrUnauthorized)
	}
	if userID == "" {
		return "", fmt.Errorf("%w: user_id is required", sharederrors.ErrInvalidArgument)
	}

	raw, err := randomRefreshToken()
	if err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}

	now := s.now()
	token := RefreshToken{
		ID:        uuid.NewString(),
		UserID:    userID,
		TokenHash: hashRefreshToken(raw),
		ExpiresAt: now.Add(s.ttl),
	}
	if err := s.store.Rotate(ctx, hashRefreshToken(oldRaw), now, token); err != nil {
		return "", err
	}
	return raw, nil
}

func randomRefreshToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
