package tokens

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/exchange-grpc/userservice/internal/domain"
	"github.com/google/uuid"
)

// RefreshTokenService управляет opaque refresh token в хранилище.
type RefreshTokenService struct {
	repo domain.RefreshTokenRepository
	ttl  time.Duration
	now  func() time.Time
}

// NewRefreshTokenService создаёт сервис refresh token.
func NewRefreshTokenService(repo domain.RefreshTokenRepository, ttl time.Duration) *RefreshTokenService {
	return &RefreshTokenService{
		repo: repo,
		ttl:  ttl,
		now:  time.Now,
	}
}

// Issue создаёт и сохраняет refresh token.
func (s *RefreshTokenService) Issue(ctx context.Context, userID string) (string, error) {
	raw, err := randomToken()
	if err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}

	now := s.now()
	token := domain.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    userID,
		TokenHash: hashToken(raw),
		ExpiresAt: now.Add(s.ttl),
	}
	if err := s.repo.Save(ctx, token); err != nil {
		return "", err
	}
	return raw, nil
}

// Validate проверяет refresh token и возвращает user_id.
func (s *RefreshTokenService) Validate(ctx context.Context, raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("%w: refresh token is required", domain.ErrUnauthorized)
	}

	stored, err := s.repo.GetByTokenHash(ctx, hashToken(raw))
	if err != nil {
		return "", err
	}
	if !stored.IsActive(s.now()) {
		return "", fmt.Errorf("%w: refresh token expired or revoked", domain.ErrUnauthorized)
	}
	return stored.UserID, nil
}

// Revoke отзывает refresh token.
func (s *RefreshTokenService) Revoke(ctx context.Context, raw string) error {
	if raw == "" {
		return fmt.Errorf("%w: refresh token is required", domain.ErrInvalidArgument)
	}

	stored, err := s.repo.GetByTokenHash(ctx, hashToken(raw))
	if err != nil {
		return err
	}
	return s.repo.Revoke(ctx, stored.ID)
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
