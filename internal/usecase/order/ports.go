package order

import "context"

// MarketChecker проверяет, что рынок существует, доступен для торговли и разрешён пользователю.
type MarketChecker interface {
	EnsureMarketAvailable(ctx context.Context, marketID string, userRoles []string) error
}
