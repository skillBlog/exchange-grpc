package domain

import "context"

// MarketRepository предоставляет доступ к данным спотовых рынков.
type MarketRepository interface {
	GetByID(ctx context.Context, id string) (Market, error)
	ListActive(ctx context.Context) ([]Market, error)
}

// OrderRepository предоставляет доступ к хранилищу ордеров.
type OrderRepository interface {
	Save(ctx context.Context, order Order) error
	GetByID(ctx context.Context, id string) (Order, error)
	GetByIDAndUserID(ctx context.Context, id, userID string) (Order, error)
}
