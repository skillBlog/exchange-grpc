package application

import (
	"context"

	"github.com/exchange-grpc/orderservice/internal/domain"
)

// MarketChecker проверяет, что рынок существует, доступен для торговли и разрешён пользователю.
type MarketChecker interface {
	EnsureMarketAvailable(ctx context.Context, marketID string, userRoles []string) error
}

// CreateOrderRateLimiter ограничивает частоту CreateOrder (глобально и per-user).
type CreateOrderRateLimiter interface {
	Allow(ctx context.Context, userID string, userRoles []string) error
}

// OrderNotifier публикует обновления статуса ордера.
type OrderNotifier interface {
	Publish(orderID string, status domain.OrderStatus)
}
