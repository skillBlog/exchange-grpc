package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Consumer читает order.events и публикует в локальный hub.
// GroupID уникален на процесс: каждая реплика получает все партиции (live-стримы).
type Consumer struct {
	reader *kafkago.Reader
	hub    application.OrderNotifier
	log    *zap.Logger
}

// NewHubConsumerGroupID — group id, который не шарится между репликами.
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
		hub: hub,
		log: log,
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
		if err := DispatchToHub(c.hub, msg.Value); err != nil {
			c.log.Error("kafka consumer skip invalid event",
				zap.String("topic", msg.Topic),
				zap.Int("partition", msg.Partition),
				zap.Int64("offset", msg.Offset),
				zap.Error(err),
			)
		}
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			c.log.Error("kafka consumer commit failed", zap.Error(err))
		}
	}
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
