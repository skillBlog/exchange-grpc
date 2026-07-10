package application

// ViewMarketsRateLimiter ограничивает частоту вызовов ViewMarkets по user_id.
type ViewMarketsRateLimiter interface {
	Allow(userID string) bool
}
