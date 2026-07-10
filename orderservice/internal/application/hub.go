package application

import (
	"sync"

	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/shared/logger"
	"go.uber.org/zap"
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
	log              *zap.Logger
}

// NewUpdateHub создаёт in-memory hub обновлений ордеров.
func NewUpdateHub(subscriberBuffer int, log *zap.Logger) *UpdateHub {
	if subscriberBuffer <= 0 {
		subscriberBuffer = defaultSubscriberBuffer
	}
	if log == nil {
		log = logger.NewNop()
	}
	return &UpdateHub{
		subscribers:      make(map[string]map[chan UpdateEvent]struct{}),
		subscriberBuffer: subscriberBuffer,
		log:              log,
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
			h.log.Warn("order update dropped: subscriber buffer full",
				zap.String("order_id", orderID),
				zap.String("status", string(status)),
			)
		}
	}
}

// Subscribe регистрирует слушателя для конкретного ордера.
// unsubscribe безопасен при повторных вызовах благодаря sync.Once.
func (h *UpdateHub) Subscribe(orderID string) (<-chan UpdateEvent, func()) {
	ch := make(chan UpdateEvent, h.subscriberBuffer)

	h.mu.Lock()
	if h.subscribers[orderID] == nil {
		h.subscribers[orderID] = make(map[chan UpdateEvent]struct{})
	}
	h.subscribers[orderID][ch] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()

			delete(h.subscribers[orderID], ch)
			if len(h.subscribers[orderID]) == 0 {
				delete(h.subscribers, orderID)
			}
			close(ch)
		})
	}

	return ch, unsubscribe
}
