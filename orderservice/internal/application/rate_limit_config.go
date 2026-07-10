package application

import "time"

// CreateOrderRateLimitConfig задаёт глобальный и per-user лимиты CreateOrder.
type CreateOrderRateLimitConfig struct {
	GlobalLimit   int
	GlobalWindow  time.Duration
	BasicLimit    int
	PremiumLimit  int
	AdminLimit    int
	UserWindow    time.Duration
}

// LimitForTier возвращает per-user лимит для tier.
func (c CreateOrderRateLimitConfig) LimitForTier(tier CreateOrderRateTier) int {
	switch tier {
	case CreateOrderRateTierAdmin:
		return c.AdminLimit
	case CreateOrderRateTierPremium:
		return c.PremiumLimit
	default:
		return c.BasicLimit
	}
}
