package application

import (
	"sync"

	"github.com/exchange-grpc/orderservice/internal/domain"
)

const defaultSubscriberBuffer = 256

// UpdateEvent отправляется подписчикам на обновления ордера.
type UpdateEvent struct {
	OrderID string
	Status  domain.OrderStatus
}

// UpdateHub рассылает изменения статуса ордера подписчикам.
type UpdateHub struct {
	mu               sync.RWMutex
	subscribers      map[string]map[chan UpdateEvent]struct{}
	subscriberBuffer int
}

// NewUpdateHub создаёт in-memory hub обновлений ордеров.
func NewUpdateHub(subscriberBuffer int) *UpdateHub {
	if subscriberBuffer <= 0 {
		subscriberBuffer = defaultSubscriberBuffer
	}
	return &UpdateHub{
		subscribers:      make(map[string]map[chan UpdateEvent]struct{}),
		subscriberBuffer: subscriberBuffer,
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
	ch := make(chan UpdateEvent, h.subscriberBuffer)

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
