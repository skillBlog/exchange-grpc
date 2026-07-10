package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/exchange-grpc/userservice/internal/domain"
	"github.com/jackc/pgx/v5"
)

// RefreshTokenRepository хранит refresh token в PostgreSQL.
type RefreshTokenRepository struct {
	db *DB
}

// NewRefreshTokenRepository создаёт postgres refresh token repository.
func NewRefreshTokenRepository(db *DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Save сохраняет refresh token.
func (r *RefreshTokenRepository) Save(ctx context.Context, token domain.RefreshToken) error {
	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked_at)
		VALUES ($1, $2, $3, $4, $5)
	`, token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.RevokedAt)
	if err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

// GetByTokenHash возвращает refresh token по хешу.
func (r *RefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (domain.RefreshToken, error) {
	row := r.db.Pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`, tokenHash)

	var token domain.RefreshToken
	var revokedAt *time.Time
	if err := row.Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &revokedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.RefreshToken{}, domain.ErrUnauthorized
		}
		return domain.RefreshToken{}, err
	}
	token.RevokedAt = revokedAt
	return token, nil
}

// Revoke помечает refresh token отозванным.
func (r *RefreshTokenRepository) Revoke(ctx context.Context, id string) error {
	tag, err := r.db.Pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
