package hub

import (
	"sync"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
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
	userSubscribers  map[string]map[chan application.UpdateEvent]struct{}
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
		userSubscribers:  make(map[string]map[chan application.UpdateEvent]struct{}),
		subscriberBuffer: subscriberBuffer,
		publishTimeout:   publishTimeout,
		log:              log,
	}
}

// Publish уведомляет подписчиков ордера и пользователя.
// Паника на одном закрытом канале не прерывает рассылку остальным.
func (h *UpdateHub) Publish(event application.UpdateEvent) {
	if h == nil {
		return
	}

	h.mu.RLock()
	chans := make([]chan application.UpdateEvent, 0, len(h.subscribers[event.OrderID])+len(h.userSubscribers[event.UserID]))
	for ch := range h.subscribers[event.OrderID] {
		chans = append(chans, ch)
	}
	if event.UserID != "" {
		for ch := range h.userSubscribers[event.UserID] {
			chans = append(chans, ch)
		}
	}
	h.mu.RUnlock()

	for _, ch := range chans {
		h.publishOne(ch, event)
	}
}

func (h *UpdateHub) publishOne(ch chan application.UpdateEvent, event application.UpdateEvent) {
	defer func() {
		if recovered := recover(); recovered != nil {
			h.log.Error("order update publish panic recovered",
				zap.String("order_id", event.OrderID),
				zap.String("user_id", event.UserID),
				zap.String("status", string(event.Status)),
				zap.Any("panic", recovered),
			)
		}
	}()

	timer := time.NewTimer(h.publishTimeout)
	defer timer.Stop()
	select {
	case ch <- event:
	case <-timer.C:
		h.log.Warn("order update timed out waiting for subscriber",
			zap.String("order_id", event.OrderID),
			zap.String("user_id", event.UserID),
			zap.String("status", string(event.Status)),
			zap.Duration("timeout", h.publishTimeout),
		)
	}
}

// Subscribe регистрирует слушателя для конкретного ордера.
func (h *UpdateHub) Subscribe(orderID string) (<-chan application.UpdateEvent, func()) {
	return h.subscribe(h.subscribers, orderID)
}

// SubscribeUser регистрирует слушателя для всех ордеров пользователя.
func (h *UpdateHub) SubscribeUser(userID string) (<-chan application.UpdateEvent, func()) {
	return h.subscribe(h.userSubscribers, userID)
}

// unsubscribe безопасен при повторных вызовах благодаря sync.Once.
func (h *UpdateHub) subscribe(store map[string]map[chan application.UpdateEvent]struct{}, key string) (<-chan application.UpdateEvent, func()) {
	ch := make(chan application.UpdateEvent, h.subscriberBuffer)

	h.mu.Lock()
	if store[key] == nil {
		store[key] = make(map[chan application.UpdateEvent]struct{})
	}
	store[key][ch] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()

			delete(store[key], ch)
			if len(store[key]) == 0 {
				delete(store, key)
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
