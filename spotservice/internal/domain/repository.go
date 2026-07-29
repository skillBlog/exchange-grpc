package domain

import "context"

// MarketRepository предоставляет доступ к данным спотовых рынков.
type MarketRepository interface {
	GetByID(ctx context.Context, id string) (Market, error)
	// ListActivePage возвращает страницу активных рынков, доступных ролям пользователя.
	// limit — максимум записей; afterID — exclusive cursor по id (пустой = с начала).
	ListActivePage(ctx context.Context, userRoles []string, limit int, afterID string) ([]Market, error)
	Ping(ctx context.Context) error
}
