package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	kafkago "github.com/segmentio/kafka-go"
)

const defaultProduceTimeout = 5 * time.Second

// Event — контракт сообщения в order.events (event_id для идемпотентных консьюмеров, G5).
type Event struct {
	EventID     string          `json:"event_id"`
	EventType   string          `json:"event_type"`
	AggregateID string          `json:"aggregate_id"`
	Payload     json.RawMessage `json:"payload"`
	CreatedAt   time.Time       `json:"created_at"`
}

// Producer пишет outbox-события в Kafka. Key = aggregate_id (order_id), чтобы события одного ордера шли в одну партицию.
type Producer struct {
	writer  *kafkago.Writer
	brokers []string
	timeout time.Duration
}

// NewProducer создаёт writer. brokers не должны быть пустыми.
func NewProducer(brokers []string, topic string, timeout time.Duration) *Producer {
	if timeout <= 0 {
		timeout = defaultProduceTimeout
	}
	if topic == "" {
		topic = "order.events"
	}
	stored := append([]string(nil), brokers...)
	return &Producer{
		brokers: stored,
		timeout: timeout,
		writer: &kafkago.Writer{
			Addr:                   kafkago.TCP(stored...),
			Topic:                  topic,
			Balancer:               &kafkago.Hash{},
			RequiredAcks:           kafkago.RequireAll,
			AllowAutoTopicCreation: true,
			BatchTimeout:           10 * time.Millisecond,
		},
	}
}

// Publish сериализует outbox-событие и пишет в Kafka.
func (p *Producer) Publish(ctx context.Context, event application.OutboxEvent) error {
	if p == nil || p.writer == nil {
		return fmt.Errorf("kafka producer is not configured")
	}
	msg, err := messageFromEvent(ctx, event)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("kafka write: %w", err)
	}
	return nil
}

// Ping проверяет, что брокер отвечает (optional health).
func (p *Producer) Ping(ctx context.Context) error {
	if p == nil || p.writer == nil {
		return fmt.Errorf("kafka producer is not configured")
	}
	if len(p.brokers) == 0 || strings.TrimSpace(p.brokers[0]) == "" {
		return fmt.Errorf("kafka brokers are empty")
	}
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", p.brokers[0])
	if err != nil {
		return fmt.Errorf("kafka dial: %w", err)
	}
	return conn.Close()
}

// Close закрывает writer.
func (p *Producer) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

// MarshalEnvelope сериализует outbox-событие в конверт order.events.
func MarshalEnvelope(event application.OutboxEvent) ([]byte, error) {
	body, err := json.Marshal(Event{
		EventID:     event.ID,
		EventType:   event.EventType,
		AggregateID: event.AggregateID,
		Payload:     event.Payload,
		CreatedAt:   event.CreatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal kafka event: %w", err)
	}
	return body, nil
}

func messageFromEvent(ctx context.Context, event application.OutboxEvent) (kafkago.Message, error) {
	body, err := MarshalEnvelope(event)
	if err != nil {
		return kafkago.Message{}, err
	}
	headers := injectTrace(ctx, []kafkago.Header{
		{Key: "event_id", Value: []byte(event.ID)},
		{Key: "event_type", Value: []byte(event.EventType)},
	})
	return kafkago.Message{
		Key:     []byte(event.AggregateID),
		Value:   body,
		Headers: headers,
	}, nil
}

var _ application.OrderEventPublisher = (*Producer)(nil)
