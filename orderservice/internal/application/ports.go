package application

import (
	"context"

	"github.com/exchange-grpc/orderservice/internal/domain"
)

// MarketChecker проверяет, что рынок существует, доступен для торговли и разрешён пользователю.
// При успехе возвращает торговые лимиты рынка (пустые поля = правило не задано).
type MarketChecker interface {
	EnsureMarketAvailable(ctx context.Context, marketID string, userRoles []string) (domain.MarketLimits, error)
}

// CreateOrderRateLimiter ограничивает частоту CreateOrder (глобально и per-user).
type CreateOrderRateLimiter interface {
	Allow(ctx context.Context, userID string, userRoles []string) error
}

// OrderNotifier публикует обновления статуса ордера.
type OrderNotifier interface {
	Publish(event UpdateEvent)
}

// OrderUpdateHub — pub/sub обновлений ордеров для streaming.
type OrderUpdateHub interface {
	OrderNotifier
	Subscribe(orderID string) (<-chan UpdateEvent, func())
	SubscribeUser(userID string) (<-chan UpdateEvent, func())
}
