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

type registration struct {
	store   map[string]subscriberSet
	key     string
	dropped chan struct{}
}

// UpdateHub — in-memory реализация рассылки обновлений ордеров.
// Канал событий не закрывается. Медленный подписчик удаляется из мапы,
// а его сигнал dropped закрывается, чтобы стрим завершился.
type UpdateHub struct {
	mu               sync.RWMutex
	subscribers      map[string]subscriberSet
	userSubscribers  map[string]subscriberSet
	regs             map[chan application.UpdateEvent]registration
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
		regs:             make(map[chan application.UpdateEvent]registration),
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
		h.drop(ch, event)
	}
}

func (h *UpdateHub) drop(ch chan application.UpdateEvent, event application.UpdateEvent) {
	dropped, ok := h.remove(ch)
	if !ok {
		return
	}
	close(dropped)
	h.log.Error("slow subscriber dropped",
		zap.String("order_id", event.OrderID),
		zap.String("user_id", event.UserID),
		zap.String("status", string(event.Status)),
		zap.Duration("timeout", h.publishTimeout),
	)
}

func (h *UpdateHub) remove(ch chan application.UpdateEvent) (chan struct{}, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	reg, ok := h.regs[ch]
	if !ok {
		return nil, false
	}
	delete(h.regs, ch)
	if set := reg.store[reg.key]; set != nil {
		delete(set, ch)
		if len(set) == 0 {
			delete(reg.store, reg.key)
		}
	}
	return reg.dropped, true
}

// Subscribe регистрирует слушателя ордера.
// Второй канал закрывается, когда хаб снимает медленного подписчика. Канал событий не закрывается.
func (h *UpdateHub) Subscribe(orderID string) (<-chan application.UpdateEvent, <-chan struct{}, func()) {
	return h.subscribe(h.subscribers, orderID, application.UserStreamFilter{})
}

// SubscribeUser регистрирует слушателя ордеров пользователя с опциональным фильтром.
// Второй канал закрывается, когда хаб снимает медленного подписчика. Канал событий не закрывается.
func (h *UpdateHub) SubscribeUser(userID string, filter application.UserStreamFilter) (<-chan application.UpdateEvent, <-chan struct{}, func()) {
	return h.subscribe(h.userSubscribers, userID, filter)
}

func (h *UpdateHub) subscribe(store map[string]subscriberSet, key string, filter application.UserStreamFilter) (<-chan application.UpdateEvent, <-chan struct{}, func()) {
	ch := make(chan application.UpdateEvent, h.subscriberBuffer)
	dropped := make(chan struct{})

	h.mu.Lock()
	if store[key] == nil {
		store[key] = make(subscriberSet)
	}
	store[key][ch] = filter
	h.regs[ch] = registration{store: store, key: key, dropped: dropped}
	h.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.remove(ch)
		})
	}

	return ch, dropped, unsubscribe
}

var (
	_ application.OrderNotifier  = (*UpdateHub)(nil)
	_ application.OrderUpdateHub = (*UpdateHub)(nil)
)
