package domain

import "context"

// OrderRepository предоставляет доступ к хранилищу ордеров.
type OrderRepository interface {
	Save(ctx context.Context, order Order) error
	GetByID(ctx context.Context, id string) (Order, error)
	GetByIDAndUserID(ctx context.Context, id, userID string) (Order, error)
}
