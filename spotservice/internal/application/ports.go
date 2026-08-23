package application

import "context"

// ViewMarketsRateLimiter ограничивает частоту вызовов ViewMarkets по user_id.
type ViewMarketsRateLimiter interface {
	Allow(ctx context.Context, userID string) error
}
