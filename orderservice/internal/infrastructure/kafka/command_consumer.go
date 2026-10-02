package kafka

import (
	"context"
	"errors"

	"github.com/exchange-grpc/orderservice/internal/application"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// DefaultCommandGroupID шарится между репликами: каждая команда обрабатывается один раз.
const DefaultCommandGroupID = "orderservice-commands"

// CommandConsumer читает order.commands, применяет через inbox и при яде пишет DLQ.
type CommandConsumer struct {
	reader      *kafkago.Reader
	topic       string
	groupID     string
	handler     application.OrderCommandHandler
	dlq         application.DeadLetterSink
	maxAttempts int
	log         *zap.Logger
}

// NewCommandConsumer создаёт reader с shared consumer group.
// Пустой groupID → DefaultCommandGroupID (не уникальный на процесс, в отличие от G5 hub).
func NewCommandConsumer(
	brokers []string,
	topic, groupID string,
	handler application.OrderCommandHandler,
	dlq application.DeadLetterSink,
	maxAttempts int,
	log *zap.Logger,
) *CommandConsumer {
	if topic == "" {
		topic = "order.commands"
	}
	if groupID == "" {
		groupID = DefaultCommandGroupID
	}
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &CommandConsumer{
		topic:       topic,
		groupID:     groupID,
		handler:     handler,
		dlq:         dlq,
		maxAttempts: maxAttempts,
		log:         log,
		reader: kafkago.NewReader(kafkago.ReaderConfig{
			Brokers:               brokers,
			Topic:                 topic,
			GroupID:               groupID,
			StartOffset:           kafkago.FirstOffset,
			MinBytes:              1,
			MaxBytes:              10 << 20,
			WatchPartitionChanges: true,
		}),
	}
}

// Run читает команды до отмены ctx. Яд уходит в DLQ, offset коммитится — партиция не стопорится.
func (c *CommandConsumer) Run(ctx context.Context) {
	if c == nil || c.reader == nil {
		return
	}
	defer func() { _ = c.reader.Close() }()

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			c.log.Error("kafka command consumer fetch failed", zap.Error(err))
			continue
		}

		for {
			if ctx.Err() != nil {
				return
			}
			err := application.ProcessCommandMessage(ctx, msg.Value, c.handler, c.dlq, c.maxAttempts)
			if err == nil {
				break
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			c.log.Error("kafka command processing failed, retrying same message",
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
			c.log.Error("kafka command consumer commit failed", zap.Error(err))
		}
	}
}

// Close закрывает reader.
func (c *CommandConsumer) Close() error {
	if c == nil || c.reader == nil {
		return nil
	}
	return c.reader.Close()
}
