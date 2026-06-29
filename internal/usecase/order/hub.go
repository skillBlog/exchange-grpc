package order

import (
	"sync"

	"github.com/exchange-grpc/internal/domain"
)

// UpdateEvent отправляется подписчикам на обновления ордера.
type UpdateEvent struct {
	OrderID string
	Status  domain.OrderStatus
}

// UpdateHub рассылает изменения статуса ордера подписчикам.
type UpdateHub struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan UpdateEvent]struct{}
}

// NewUpdateHub создаёт in-memory hub обновлений ордеров.
func NewUpdateHub() *UpdateHub {
	return &UpdateHub{
		subscribers: make(map[string]map[chan UpdateEvent]struct{}),
	}
}

// Publish уведомляет подписчиков о новом статусе ордера.
func (h *UpdateHub) Publish(orderID string, status domain.OrderStatus) {
	if h == nil {
		return
	}

	event := UpdateEvent{OrderID: orderID, Status: status}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.subscribers[orderID] {
		select {
		case ch <- event:
		default:
		}
	}
}

// Subscribe регистрирует слушателя для конкретного ордера.
func (h *UpdateHub) Subscribe(orderID string) (<-chan UpdateEvent, func()) {
	ch := make(chan UpdateEvent, 8)

	h.mu.Lock()
	if h.subscribers[orderID] == nil {
		h.subscribers[orderID] = make(map[chan UpdateEvent]struct{})
	}
	h.subscribers[orderID][ch] = struct{}{}
	h.mu.Unlock()

	unsubscribe := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		delete(h.subscribers[orderID], ch)
		close(ch)
		if len(h.subscribers[orderID]) == 0 {
			delete(h.subscribers, orderID)
		}
	}

	return ch, unsubscribe
}
