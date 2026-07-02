package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/exchange-grpc/orderservice/internal/domain"
)

// OrderRepository хранит ордера в памяти.
type OrderRepository struct {
	mu     sync.RWMutex
	orders map[string]domain.Order
}

// NewOrderRepository создаёт пустой in-memory репозиторий ордеров.
func NewOrderRepository() *OrderRepository {
	return &OrderRepository{
		orders: make(map[string]domain.Order),
	}
}

// Save сохраняет ордер по идентификатору.
func (r *OrderRepository) Save(_ context.Context, order domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID] = order
	return nil
}

// GetByID возвращает ордер по идентификатору.
func (r *OrderRepository) GetByID(_ context.Context, id string) (domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, ok := r.orders[id]
	if !ok {
		return domain.Order{}, fmt.Errorf("%w: order %q", domain.ErrNotFound, id)
	}
	return order, nil
}

// GetByIDAndUserID возвращает ордер при совпадении order_id и user_id.
func (r *OrderRepository) GetByIDAndUserID(_ context.Context, orderID, userID string) (domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, ok := r.orders[orderID]
	if !ok || order.UserID != userID {
		return domain.Order{}, fmt.Errorf("%w: order %q", domain.ErrNotFound, orderID)
	}
	return order, nil
}

var _ domain.OrderRepository = (*OrderRepository)(nil)
