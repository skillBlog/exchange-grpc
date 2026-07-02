package domain

import "context"

// MarketRepository предоставляет доступ к данным спотовых рынков.
type MarketRepository interface {
	GetByID(ctx context.Context, id string) (Market, error)
	ListActive(ctx context.Context) ([]Market, error)
}
