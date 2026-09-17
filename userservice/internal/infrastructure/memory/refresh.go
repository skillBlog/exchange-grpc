package memory

import "github.com/exchange-grpc/shared/sessionvalidation"

// NewRefreshTokenRepository возвращает in-memory RefreshTokenStore для тестов.
func NewRefreshTokenRepository() sessionvalidation.RefreshTokenStore {
	return sessionvalidation.NewMemoryRefreshTokenStore()
}
