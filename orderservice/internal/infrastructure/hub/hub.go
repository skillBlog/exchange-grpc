package hub

import (
	"sync"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/shared/logger"
	"go.uber.org/zap"
)

const (
	defaultSubscriberBuffer = 256
	defaultPublishTimeout   = 100 * time.Millisecond
)

// UpdateHub — in-memory реализация рассылки обновлений ордеров.
type UpdateHub struct {
	mu               sync.RWMutex
	subscribers      map[string]map[chan application.UpdateEvent]struct{}
	subscriberBuffer int
	publishTimeout   time.Duration
	log              *zap.Logger
}

// NewUpdateHub создаёт in-memory hub обновлений ордеров.
// publishTimeout — сколько ждать медленного подписчика; при <= 0 используется 100ms.
func NewUpdateHub(subscriberBuffer int, log *zap.Logger, publishTimeout time.Duration) *UpdateHub {
	if subscriberBuffer <= 0 {
		subscriberBuffer = defaultSubscriberBuffer
	}
	if publishTimeout <= 0 {
		publishTimeout = defaultPublishTimeout
	}
	if log == nil {
		log = logger.NewNop()
	}
	return &UpdateHub{
		subscribers:      make(map[string]map[chan application.UpdateEvent]struct{}),
		subscriberBuffer: subscriberBuffer,
		publishTimeout:   publishTimeout,
		log:              log,
	}
}

// Publish уведомляет подписчиков о новом статусе ордера.
// Блокирующая отправка с timeout; panic внутри recover'ится, чтобы не ронять CreateOrder.
func (h *UpdateHub) Publish(orderID string, status domain.OrderStatus, updatedAt time.Time) {
	if h == nil {
		return
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			h.log.Error("order update publish panic recovered",
				zap.String("order_id", orderID),
				zap.String("status", string(status)),
				zap.Any("panic", recovered),
			)
		}
	}()

	event := application.UpdateEvent{OrderID: orderID, Status: status, UpdatedAt: updatedAt}

	h.mu.RLock()
	chans := make([]chan application.UpdateEvent, 0, len(h.subscribers[orderID]))
	for ch := range h.subscribers[orderID] {
		chans = append(chans, ch)
	}
	h.mu.RUnlock()

	for _, ch := range chans {
		timer := time.NewTimer(h.publishTimeout)
		select {
		case ch <- event:
			timer.Stop()
		case <-timer.C:
			h.log.Warn("order update timed out waiting for subscriber",
				zap.String("order_id", orderID),
				zap.String("status", string(status)),
				zap.Duration("timeout", h.publishTimeout),
			)
		}
	}
}

// Subscribe регистрирует слушателя для конкретного ордера.
// unsubscribe безопасен при повторных вызовах благодаря sync.Once.
func (h *UpdateHub) Subscribe(orderID string) (<-chan application.UpdateEvent, func()) {
	ch := make(chan application.UpdateEvent, h.subscriberBuffer)

	h.mu.Lock()
	if h.subscribers[orderID] == nil {
		h.subscribers[orderID] = make(map[chan application.UpdateEvent]struct{})
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

var (
	_ application.OrderNotifier  = (*UpdateHub)(nil)
	_ application.OrderUpdateHub = (*UpdateHub)(nil)
)
