package memory

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/exchange-grpc/orderservice/internal/domain"
)

// OrderRepository хранит ордера в памяти для тестов.
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

// Create сохраняет новый ордер.
func (r *OrderRepository) Create(_ context.Context, order domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.orders[order.ID]; exists {
		return fmt.Errorf("%w: order %q already exists", domain.ErrAlreadyExists, order.ID)
	}
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
	if !ok {
		return domain.Order{}, fmt.Errorf("%w: order %q", domain.ErrNotFound, orderID)
	}
	if order.UserID != userID {
		return domain.Order{}, domain.ErrForbidden
	}
	return order, nil
}

// ListByUserID возвращает страницу ордеров пользователя по курсору id.
func (r *OrderRepository) ListByUserID(_ context.Context, userID string, limit int, afterID string) ([]domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		return []domain.Order{}, nil
	}

	result := make([]domain.Order, 0)
	for _, order := range r.orders {
		if order.UserID != userID {
			continue
		}
		if afterID != "" && strings.Compare(order.ID, afterID) <= 0 {
			continue
		}
		result = append(result, order)
	}

	slices.SortFunc(result, func(a, b domain.Order) int {
		return strings.Compare(a.ID, b.ID)
	})

	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// UpdateStatus обновляет статус ордера.
func (r *OrderRepository) UpdateStatus(_ context.Context, id string, status domain.OrderStatus, updatedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	order, ok := r.orders[id]
	if !ok {
		return fmt.Errorf("%w: order %q", domain.ErrNotFound, id)
	}
	order.Status = status
	order.UpdatedAt = updatedAt
	r.orders[id] = order
	return nil
}

// IdempotencyStore хранит idempotency keys в памяти.
type IdempotencyStore struct {
	mu   sync.RWMutex
	keys map[string]string
}

// NewIdempotencyStore создаёт in-memory idempotency store.
func NewIdempotencyStore() *IdempotencyStore {
	return &IdempotencyStore{keys: make(map[string]string)}
}

func idempotencyMapKey(userID, key string) string {
	return userID + ":" + key
}

// GetOrderID возвращает order_id по idempotency key.
func (s *IdempotencyStore) GetOrderID(_ context.Context, userID, key string) (string, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	orderID, ok := s.keys[idempotencyMapKey(userID, key)]
	return orderID, ok, nil
}

// Reserve резервирует idempotency key под orderID.
func (s *IdempotencyStore) Reserve(_ context.Context, userID, key, orderID string) (bool, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	mapKey := idempotencyMapKey(userID, key)
	if existing, ok := s.keys[mapKey]; ok {
		return false, existing, nil
	}
	s.keys[mapKey] = orderID
	return true, "", nil
}

// Release снимает резерв idempotency key.
func (s *IdempotencyStore) Release(_ context.Context, userID, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.keys, idempotencyMapKey(userID, key))
	return nil
}

var (
	_ domain.OrderRepository  = (*OrderRepository)(nil)
	_ domain.IdempotencyStore = (*IdempotencyStore)(nil)
)
