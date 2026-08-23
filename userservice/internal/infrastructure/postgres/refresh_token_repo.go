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
// Идемпотентно: повторный revoke или уже отозванный токен не считаются ошибкой.
func (r *RefreshTokenRepository) Revoke(ctx context.Context, id string) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

// Rotate атомарно отзывает старый refresh token и сохраняет новый.
func (r *RefreshTokenRepository) Rotate(ctx context.Context, oldTokenHash string, now time.Time, newToken domain.RefreshToken) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin rotate refresh token: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
		FOR UPDATE
	`, oldTokenHash)

	var stored domain.RefreshToken
	var revokedAt *time.Time
	if err := row.Scan(&stored.ID, &stored.UserID, &stored.TokenHash, &stored.ExpiresAt, &revokedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrUnauthorized
		}
		return err
	}
	stored.RevokedAt = revokedAt
	if !stored.IsActive(now) || stored.UserID != newToken.UserID {
		return domain.ErrUnauthorized
	}

	tag, err := tx.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = $1
		WHERE id = $2 AND revoked_at IS NULL
	`, now, stored.ID)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUnauthorized
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked_at)
		VALUES ($1, $2, $3, $4, $5)
	`, newToken.ID, newToken.UserID, newToken.TokenHash, newToken.ExpiresAt, newToken.RevokedAt)
	if err != nil {
		return fmt.Errorf("insert rotated refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit rotate refresh token: %w", err)
	}
	return nil
}
