package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	kafkago "github.com/segmentio/kafka-go"
)

const maxDLQReasonBytes = 512

// DeadLetterPublisher пишет исходный payload в DLQ-топик (G6).
type DeadLetterPublisher struct {
	writer  *kafkago.Writer
	timeout time.Duration
}

// NewDeadLetterPublisher создаёт writer в order.commands.dlq.
func NewDeadLetterPublisher(brokers []string, topic string, timeout time.Duration) *DeadLetterPublisher {
	if timeout <= 0 {
		timeout = defaultProduceTimeout
	}
	if topic == "" {
		topic = "order.commands.dlq"
	}
	return &DeadLetterPublisher{
		timeout: timeout,
		writer: &kafkago.Writer{
			Addr:                   kafkago.TCP(brokers...),
			Topic:                  topic,
			Balancer:               &kafkago.LeastBytes{},
			RequiredAcks:           kafkago.RequireAll,
			AllowAutoTopicCreation: true,
			BatchTimeout:           10 * time.Millisecond,
		},
	}
}

// Publish кладёт original value в DLQ; reason — в header dlq-reason.
func (p *DeadLetterPublisher) Publish(ctx context.Context, original []byte, reason string) error {
	if p == nil || p.writer == nil {
		return fmt.Errorf("kafka dlq is not configured")
	}
	if reason == "" {
		reason = "unknown"
	}
	if len(reason) > maxDLQReasonBytes {
		reason = reason[:maxDLQReasonBytes]
	}
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	if err := p.writer.WriteMessages(ctx, kafkago.Message{
		Value: original,
		Headers: []kafkago.Header{
			{Key: "dlq-reason", Value: []byte(reason)},
		},
	}); err != nil {
		return fmt.Errorf("kafka dlq write: %w", err)
	}
	return nil
}

// Close закрывает writer.
func (p *DeadLetterPublisher) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

var _ application.DeadLetterSink = (*DeadLetterPublisher)(nil)
