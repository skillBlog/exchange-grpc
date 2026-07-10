package domain

import "time"

// RefreshToken — сохранённый refresh token пользователя.
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
