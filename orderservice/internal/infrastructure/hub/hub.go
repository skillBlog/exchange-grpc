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

type subscriberSet map[chan application.UpdateEvent]application.UserStreamFilter

// UpdateHub — in-memory реализация рассылки обновлений ордеров.
// Каналы не закрываются: вызывающий делает unsubscribe, хаб безопасно публикует
// при гонке с отпиской (см. контракт OrderUpdateHub).
type UpdateHub struct {
	mu               sync.RWMutex
	subscribers      map[string]subscriberSet
	userSubscribers  map[string]subscriberSet
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
		subscribers:      make(map[string]subscriberSet),
		userSubscribers:  make(map[string]subscriberSet),
		subscriberBuffer: subscriberBuffer,
		publishTimeout:   publishTimeout,
		log:              log,
	}
}

// Publish уведомляет подписчиков ордера и пользователя.
// Снимок каналов берётся под RLock; отписка после снимка безопасна — канал не закрыт.
// Паника на закрытом канале (нарушение контракта) перехватывается и не рвёт рассылку.
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
		for ch, filter := range h.userSubscribers[event.UserID] {
			if !filter.Matches(event) {
				continue
			}
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

// Subscribe регистрирует слушателя ордера. Вторая функция — unsubscribe; канал не закрывается.
func (h *UpdateHub) Subscribe(orderID string) (<-chan application.UpdateEvent, func()) {
	return h.subscribe(h.subscribers, orderID, application.UserStreamFilter{})
}

// SubscribeUser регистрирует слушателя ордеров пользователя с опциональным фильтром.
// Вторая функция — unsubscribe; канал не закрывается.
func (h *UpdateHub) SubscribeUser(userID string, filter application.UserStreamFilter) (<-chan application.UpdateEvent, func()) {
	return h.subscribe(h.userSubscribers, userID, filter)
}

func (h *UpdateHub) subscribe(store map[string]subscriberSet, key string, filter application.UserStreamFilter) (<-chan application.UpdateEvent, func()) {
	ch := make(chan application.UpdateEvent, h.subscriberBuffer)

	h.mu.Lock()
	if store[key] == nil {
		store[key] = make(subscriberSet)
	}
	store[key][ch] = filter
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
		})
	}

	return ch, unsubscribe
}

var (
	_ application.OrderNotifier  = (*UpdateHub)(nil)
	_ application.OrderUpdateHub = (*UpdateHub)(nil)
)
