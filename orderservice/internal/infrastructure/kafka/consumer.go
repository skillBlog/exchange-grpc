package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/shared/tracing"
	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

const (
	defaultSeenEvents     = 10_000
	hubApplyRetryInterval = 100 * time.Millisecond
)

// Consumer читает order.events в локальный hub.
// Live-рассылка между репликами идёт через Redis pub/sub.
// Группа orderservice-hub-<uuid> и LastOffset процессом не запускаются.
type Consumer struct {
	reader *kafkago.Reader
	hub    application.OrderNotifier
	log    *zap.Logger
	seen   *eventSeen
}

// NewHubConsumerGroupID — прежний уникальный group id хаба.
// Для fan-out он не используется: каждая такая группа писала бы свой offset.
func NewHubConsumerGroupID() string {
	return "orderservice-hub-" + uuid.NewString()
}

// NewConsumer создаёт reader. groupID пустой — генерируется NewHubConsumerGroupID.
func NewConsumer(brokers []string, topic, groupID string, hub application.OrderNotifier, log *zap.Logger) *Consumer {
	if topic == "" {
		topic = "order.events"
	}
	if groupID == "" {
		groupID = NewHubConsumerGroupID()
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &Consumer{
		hub:  hub,
		log:  log,
		seen: newEventSeen(defaultSeenEvents),
		reader: kafkago.NewReader(kafkago.ReaderConfig{
			Brokers:               brokers,
			Topic:                 topic,
			GroupID:               groupID,
			StartOffset:           kafkago.LastOffset,
			MinBytes:              1,
			MaxBytes:              10 << 20,
			WatchPartitionChanges: true,
		}),
	}
}

// Run читает Kafka до отмены ctx и кладёт события в hub.
// Offset коммитится только после успешного apply. Ошибка применения
// повторяет то же сообщение и не коммитит offset, чтобы оно не потерялось.
func (c *Consumer) Run(ctx context.Context) {
	if c == nil || c.reader == nil || c.hub == nil {
		return
	}
	defer func() { _ = c.reader.Close() }()

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			c.log.Error("kafka consumer fetch failed", zap.Error(err))
			continue
		}
		if err := c.apply(ctx, msg); err != nil {
			return
		}
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			c.log.Error("kafka consumer commit failed", zap.Error(err))
		}
	}
}

// apply повторяет одно сообщение, пока оно не применится или ctx не отменят.
// nil значит, что offset можно коммитить.
func (c *Consumer) apply(ctx context.Context, msg kafkago.Message) error {
	for {
		err := c.consume(ctx, msg)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		c.logger().Error("kafka consumer apply failed, offset not committed",
			zap.String("topic", msg.Topic),
			zap.Int("partition", msg.Partition),
			zap.Int64("offset", msg.Offset),
			zap.Error(err),
		)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(hubApplyRetryInterval):
		}
	}
}

func (c *Consumer) consume(ctx context.Context, msg kafkago.Message) error {
	ctx = extractTrace(ctx, msg.Headers)
	var err error
	_, span := tracing.Start(ctx, "kafka.consume order.events",
		tracing.Attr("messaging.destination", msg.Topic),
	)
	defer tracing.End(span, &err)

	err = deliver(c.hub, c.events(), msg.Value)
	return err
}

// HubForwarder публикует конверт order.events в локальный hub один раз на event_id.
type HubForwarder struct {
	hub  application.OrderNotifier
	seen *eventSeen
}

// NewHubForwarder создаёт forwarder для Redis-подписчика и локального fallback.
func NewHubForwarder(hub application.OrderNotifier) *HubForwarder {
	return &HubForwarder{hub: hub, seen: newEventSeen(defaultSeenEvents)}
}

// Deliver разбирает payload и публикует его в hub. Повтор того же event_id пропускается.
func (f *HubForwarder) Deliver(payload []byte) error {
	if f == nil {
		return fmt.Errorf("hub forwarder is not configured")
	}
	return deliver(f.hub, f.seen, payload)
}

func deliver(hub application.OrderNotifier, seen *eventSeen, payload []byte) error {
	var envelope Event
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return fmt.Errorf("decode kafka event: %w", err)
	}
	if seen.has(envelope.EventID) {
		return nil
	}
	if err := DispatchToHub(hub, payload); err != nil {
		return err
	}
	seen.mark(envelope.EventID)
	return nil
}

func (c *Consumer) logger() *zap.Logger {
	if c == nil || c.log == nil {
		return zap.NewNop()
	}
	return c.log
}

func (c *Consumer) events() *eventSeen {
	if c.seen == nil {
		c.seen = newEventSeen(defaultSeenEvents)
	}
	return c.seen
}

// eventSeen помнит последние event_id, чтобы повтор не ушёл в hub второй раз.
type eventSeen struct {
	max   int
	ids   map[string]struct{}
	order []string
}

func newEventSeen(max int) *eventSeen {
	if max <= 0 {
		max = defaultSeenEvents
	}
	return &eventSeen{
		max: max,
		ids: make(map[string]struct{}),
	}
}

func (s *eventSeen) has(id string) bool {
	if s == nil || id == "" {
		return false
	}
	_, ok := s.ids[id]
	return ok
}

func (s *eventSeen) mark(id string) {
	if s == nil || id == "" {
		return
	}
	if _, ok := s.ids[id]; ok {
		return
	}
	s.ids[id] = struct{}{}
	s.order = append(s.order, id)
	if len(s.order) <= s.max {
		return
	}
	oldest := s.order[0]
	s.order = s.order[1:]
	delete(s.ids, oldest)
}

// Close закрывает reader.
func (c *Consumer) Close() error {
	if c == nil || c.reader == nil {
		return nil
	}
	return c.reader.Close()
}

// DispatchToHub разбирает конверт G4 и публикует UpdateEvent в hub.
func DispatchToHub(hub application.OrderNotifier, value []byte) error {
	var envelope Event
	if err := json.Unmarshal(value, &envelope); err != nil {
		return fmt.Errorf("decode kafka event: %w", err)
	}
	update, err := application.ParseOutboxUpdateEvent(application.OutboxEvent{
		ID:          envelope.EventID,
		AggregateID: envelope.AggregateID,
		EventType:   envelope.EventType,
		Payload:     envelope.Payload,
		CreatedAt:   envelope.CreatedAt,
	})
	if err != nil {
		return err
	}
	if hub != nil {
		hub.Publish(update)
	}
	return nil
}
